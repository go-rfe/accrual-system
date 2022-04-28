package server

import (
	"accrual-system/internal/models"
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
	requestTimeout = 3 * time.Second
)

func getOrderHandler(storage repository.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
		defer cancel()

		orderID := c.Param("number")
		if err := models.ValidateOrder(orderID); err != nil {
			http.Error(c.Writer, fmt.Sprintf("bad order number: %s (%q)", orderID, err), http.StatusUnprocessableEntity)
			return
		}

		order, err := storage.GetOrder(ctx, orderID)
		switch {
		case errors.Is(err, models.ErrOrderDoesntExist):
			http.Error(c.Writer, "couldn't found order", http.StatusNotFound)
			return

		case err != nil:
			http.Error(c.Writer, fmt.Sprintf("couldn't get order: %q", err), http.StatusInternalServerError)
			return
		}

		c.Writer.Header().Set("Content-Type", "application/json")
		err = models.WriteJSON(&order, c.Writer)
		if err != nil {
			log.Error().Err(err).Msg("cannot send request")
		}
	}
}

func updateOrdersHandler(storage repository.Storage, signal chan struct{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
		defer cancel()

		var order models.Order

		err := json.NewDecoder(c.Request.Body).Decode(&order)
		if err != nil {
			http.Error(c.Writer, fmt.Sprintf("can't decode provided data: %q", err), http.StatusBadRequest)
			return
		}

		err = models.ValidateOrder(order.Order)
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusBadRequest)
			return
		}

		err = storage.CreateOrder(ctx, order)
		if errors.Is(err, models.ErrOrderExists) {
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

func updateGoodsHandler(storage repository.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
		defer cancel()

		var reward models.Reward
		err := json.NewDecoder(c.Request.Body).Decode(&reward)
		if err != nil || errors.Is(models.ValidateReward(reward), models.ErrBadReward) {
			http.Error(c.Writer, fmt.Sprintf("Cannot decode provided data: %q", err), http.StatusBadRequest)
			return
		}

		err = storage.CreateReward(ctx, reward)
		if errors.Is(err, models.ErrRewardExists) {
			http.Error(c.Writer, err.Error(), http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
