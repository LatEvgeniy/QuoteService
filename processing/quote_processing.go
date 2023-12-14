package processing

import (
	"QuoteService/proto"
	"context"
	"encoding/json"
	"fmt"
)

var (
	quotesKey = "quotes"

	notFoundQuotesErrMsg = "Not found quotes for pair: %s"
)

func (q *QuoteProcessing) UpdateQuotes(pair *proto.OrderPair, price, volume float64) (*proto.QuotesEvent, error) {
	if err := q.checkQuotesExist(); err != nil {
		return nil, err
	}

	if err := q.updateRedisQuotes(pair, price, volume); err != nil {
		return nil, err
	}

	return q.GetQuotesEvent()
}

func (q *QuoteProcessing) updateRedisQuotes(pair *proto.OrderPair, price, volume float64) error {
	for stringPair := range q.RedisClient.HGetAll(context.Background(), quotesKey).Val() {
		if stringPair != pair.String() {
			continue
		}

		volumeByPriceJson, err := json.Marshal(&proto.VolumeByPrice{Price: price, Volume: volume})
		if err != nil {
			return err
		}

		if err := q.RedisClient.HSet(context.Background(), quotesKey, stringPair, volumeByPriceJson).Err(); err != nil {
			return err
		}

		return nil
	}
	return fmt.Errorf(notFoundQuotesErrMsg, pair)
}

func (q *QuoteProcessing) GetQuotesEvent() (*proto.QuotesEvent, error) {
	if err := q.checkQuotesExist(); err != nil {
		return nil, err
	}

	var event proto.QuotesEvent
	for stringPair, quote := range q.RedisClient.HGetAll(context.Background(), quotesKey).Val() {
		pairValue, exists := proto.OrderPair_value[stringPair]
		if !exists {
			return nil, fmt.Errorf(invalidOrderPairErrMsg, stringPair)
		}

		var volumeByPrice proto.VolumeByPrice
		if err := json.Unmarshal([]byte(quote), &volumeByPrice); err != nil {
			return nil, err
		}

		event.CurrentQuotes = append(event.CurrentQuotes, &proto.PairQuote{Pair: proto.OrderPair(pairValue), Price: volumeByPrice.Price, Volume: volumeByPrice.Volume})
	}

	return &event, nil
}

func (q *QuoteProcessing) checkQuotesExist() error {
	quotesByPair, err := q.RedisClient.HGetAll(context.Background(), quotesKey).Result()
	if err != nil {
		return err
	}
	if len(quotesByPair) != 0 {
		return nil
	}

	for stringPair := range proto.OrderPair_value {
		volumeByPriceJson, err := json.Marshal(&proto.VolumeByPrice{Price: 0, Volume: 0})
		if err != nil {
			return err
		}

		if err := q.RedisClient.HSet(context.Background(), quotesKey, stringPair, volumeByPriceJson).Err(); err != nil {
			return err
		}
	}

	return nil
}
