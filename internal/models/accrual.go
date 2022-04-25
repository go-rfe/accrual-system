package models

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/go-rfe/utils/luhn"
	"github.com/shopspring/decimal"
)

const (
	orderNumberBase    = 10
	orderNumberBitSize = 64
	RegisteredStatus   = "REGISTERED"
	ProcessingStatus   = "PROCESSING"
	InvalidStatus      = "INVALID"
	ProcessedStatus    = "PROCESSED"
	PercentsRewardType = "%"
	PointsRewardType   = "pt"
	OneHundredPercents = 100
)

var (
	ErrOrderExists         = errors.New("order already exists")
	ErrRewardExists        = errors.New("reward match key already exists")
	ErrOrderDoesntExist    = errors.New("order doesn't exists")
	ErrInvalidOrderNumber  = errors.New("order number is invalid")
	ErrBadReward           = errors.New("match cannot be empty string or bad reward types")
	WithoutAccrualStatuses = map[string]struct{}{
		"INVALID":    {},
		"REGISTERED": {},
		"PROCESSING": {},
	}
)

type Good struct {
	Description string          `json:"description"`
	Price       decimal.Decimal `json:"price"`
}

type Order struct {
	Order   string          `json:"order"`
	Status  string          `json:"status"`
	Accrual decimal.Decimal `json:"accrual,omitempty"`
	Goods   []Good          `json:"goods,omitempty"`
}

type Reward struct {
	Match      string          `json:"match"`
	Reward     decimal.Decimal `json:"reward"`
	RewardType string          `json:"reward_type"`
}

func ValidateOrder(number string) error {
	numberInt, err := strconv.ParseInt(number, orderNumberBase, orderNumberBitSize)
	if err != nil {
		return err
	}

	if !luhn.Valid(numberInt) {
		return ErrInvalidOrderNumber
	}

	return nil
}

func ValidateReward(reward Reward) error {
	if !(len(reward.Match) > 0) {
		return ErrBadReward
	}
	if reward.RewardType != PercentsRewardType && reward.RewardType != PointsRewardType {
		return ErrBadReward
	}
	if reward.Reward.LessThan(decimal.Zero) {
		return ErrBadReward
	}

	return nil
}

func WriteJSON(data interface{}, w io.Writer) error {
	jsonEncoder := json.NewEncoder(w)
	decimal.MarshalJSONWithoutQuotes = true
	return jsonEncoder.Encode(data)
}
