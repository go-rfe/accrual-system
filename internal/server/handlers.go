package server

import (
	"accrual-system/internal/pkg/accrual"
	"accrual-system/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-rfe/logging/log"
)

const (
	requestTimeout = 1 * time.Second
)

func getOrderHandler(store repository.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
		defer cancel()

		orderID := c.Param("number")
		if err := accrual.ValidateOrder(orderID); err != nil {
			http.Error(c.Writer, fmt.Sprintf("bad order number: %s (%q)", orderID, err), http.StatusUnprocessableEntity)
			return
		}

		order, err := store.GetOrder(ctx, orderID)
		switch {
		case errors.Is(err, accrual.ErrOrderDoesntExist):
			http.Error(c.Writer, "couldn't found order", http.StatusNotFound)
			return

		case err != nil:
			http.Error(c.Writer, fmt.Sprintf("couldn't get order: %q", err), http.StatusInternalServerError)
			return
		}

		c.Writer.Header().Set("Content-Type", "application/json")
		err = accrual.WriteJSON(&order, c.Writer)
		if err != nil {
			log.Error().Err(err).Msg("cannot send request")
		}
	}
}

func updateOrdersHandler(store repository.Storage, signal chan struct{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		createContext, createCancel := context.WithTimeout(c.Request.Context(), requestTimeout)
		defer createCancel()

		var order accrual.Order
		err := json.NewDecoder(c.Request.Body).Decode(&order)
		if err != nil {
			http.Error(c.Writer, fmt.Sprintf("can't decode provided data: %q", err), http.StatusBadRequest)
			return
		}

		err = accrual.ValidateOrder(order.Order)
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusBadRequest)
			return
		}

		err = store.CreateOrder(createContext, order)
		if errors.Is(err, accrual.ErrOrderExists) {
			http.Error(c.Writer, err.Error(), http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Writer.WriteHeader(http.StatusAccepted)

		signal <- struct{}{}
	}
}

func updateGoodsHandler(store repository.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		createContext, createCancel := context.WithTimeout(c.Request.Context(), requestTimeout)
		defer createCancel()

		var reward accrual.Reward
		err := json.NewDecoder(c.Request.Body).Decode(&reward)
		if err != nil || errors.Is(accrual.ValidateReward(reward), accrual.ErrBadReward) {
			http.Error(c.Writer, fmt.Sprintf("Cannot decode provided data: %q", err), http.StatusBadRequest)
			return
		}

		err = store.CreateReward(createContext, reward)
		if errors.Is(err, accrual.ErrRewardExists) {
			http.Error(c.Writer, err.Error(), http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
