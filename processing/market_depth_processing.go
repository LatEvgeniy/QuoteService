package processing

import (
	"QuoteService/proto"
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type QuoteProcessing struct {
	RedisClient *redis.Client
}

var (
	marketDepthKey = "marketdepth:%s"

	invalidOrderPairErrMsg      = "invalid order pair: %s"
	invalidOrderDirectionErrMsg = "invalid order direction: %s"
)

func (q *QuoteProcessing) UpdateMarketDepth(direction, pair string, price, volume float64) (*proto.MarketDepthEvent, error) {
	if err := q.checkMarketDepthExist(); err != nil {
		return nil, err
	}

	currentVolumeByPriceSlice, err := q.getCurrentVolumeByPriceSlice(direction, pair)
	if err != nil {
		return nil, err
	}

	actualVolumeByPrice := q.addVolumeToPrice(currentVolumeByPriceSlice, price, volume)
	newVolumeByPriceSlice := q.removeZeroVolume(actualVolumeByPrice)

	volumeByPriceJson, err := json.Marshal(newVolumeByPriceSlice)
	if err != nil {
		return nil, err
	}

	if err = q.RedisClient.HSet(context.Background(), fmt.Sprintf(marketDepthKey, direction), pair, volumeByPriceJson).Err(); err != nil {
		return nil, err
	}

	return q.GetMarketDepthEvent()
}

func (q *QuoteProcessing) GetMarketDepthEvent() (*proto.MarketDepthEvent, error) {
	if err := q.checkMarketDepthExist(); err != nil {
		return nil, err
	}

	var marketDepthEvent proto.MarketDepthEvent
	for stringDirection := range proto.OrderDirection_value {
		for stringPair := range proto.OrderPair_value {
			currentVolumeByPriceSlice, err := q.getCurrentVolumeByPriceSlice(stringDirection, stringPair)
			if err != nil {
				return nil, err
			}

			if currentVolumeByPriceSlice == nil {
				continue
			}

			pairValue, exists := proto.OrderPair_value[stringPair]
			if !exists {
				return nil, fmt.Errorf(invalidOrderPairErrMsg, stringPair)
			}
			directionValue, exists := proto.OrderDirection_value[stringDirection]
			if !exists {
				return nil, fmt.Errorf(invalidOrderDirectionErrMsg, stringDirection)
			}

			marketDepthEvent.MarketDepth = append(marketDepthEvent.MarketDepth, &proto.PairMatketDepth{
				Pair:          proto.OrderPair(pairValue),
				Direction:     proto.OrderDirection(directionValue),
				VolumeByPrice: currentVolumeByPriceSlice,
			})
		}
	}

	return &marketDepthEvent, nil
}

func (q *QuoteProcessing) getCurrentVolumeByPriceSlice(direction, pair string) ([]*proto.VolumeByPrice, error) {
	currentByteVolumeByPrice, err := q.RedisClient.HGet(context.Background(), fmt.Sprintf(marketDepthKey, direction), pair).Result()
	if err != nil {
		return nil, err
	}

	if currentByteVolumeByPrice == "{}" || currentByteVolumeByPrice == "[]" {
		return nil, nil
	}

	var currentVolumeByPriceSlice []*proto.VolumeByPrice
	if err = json.Unmarshal([]byte(currentByteVolumeByPrice), &currentVolumeByPriceSlice); err != nil {
		return nil, err
	}

	return currentVolumeByPriceSlice, nil
}

func (q *QuoteProcessing) addVolumeToPrice(volumeByPriceSlice []*proto.VolumeByPrice, price, volume float64) []*proto.VolumeByPrice {
	for i, volumeByPrice := range volumeByPriceSlice {
		if volumeByPrice.Price != price {
			continue
		}
		volumeByPriceSlice[i].Volume += volume
		return volumeByPriceSlice
	}

	return append(volumeByPriceSlice, &proto.VolumeByPrice{Price: price, Volume: volume})
}

func (q *QuoteProcessing) removeZeroVolume(volumeByPriceSlice []*proto.VolumeByPrice) []*proto.VolumeByPrice {
	filteredData := []*proto.VolumeByPrice{}
	for _, volumeByPrice := range volumeByPriceSlice {
		if volumeByPrice.Volume != 0 {
			filteredData = append(filteredData, volumeByPrice)
		}
	}
	return filteredData
}

func (q *QuoteProcessing) checkMarketDepthExist() error {
	for _, directionName := range proto.OrderDirection_name {
		quotesByPair, err := q.RedisClient.HGetAll(context.Background(), fmt.Sprintf(marketDepthKey, directionName)).Result()
		if err != nil {
			return err
		}
		if len(quotesByPair) != 0 {
			continue
		}
		for stringPair := range proto.OrderPair_value {
			volumeByPriceJson, err := json.Marshal(&proto.VolumeByPrice{Price: 0, Volume: 0})
			if err != nil {
				return err
			}

			if err := q.RedisClient.HSet(context.Background(), fmt.Sprintf(marketDepthKey, directionName), stringPair, volumeByPriceJson).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}
