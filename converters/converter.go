package converters

import (
	"QuoteService/models"
	"QuoteService/proto"
)

func ConvertPairMarketDepthToModel(protoPairMarketDepth *proto.PairMatketDepth) *models.PairMarketDepthModel {
	volumeByPriceModels := []models.VolumeByPriceModel{}
	for _, volumevolumeByPriceProto := range protoPairMarketDepth.VolumeByPrice {
		volumeByPriceModels = append(volumeByPriceModels, models.VolumeByPriceModel{
			Price:  volumevolumeByPriceProto.Price,
			Volume: volumevolumeByPriceProto.Volume,
		})
	}
	return &models.PairMarketDepthModel{
		OrderPair:           protoPairMarketDepth.Pair.String(),
		OrderDirection:      protoPairMarketDepth.Direction.String(),
		VolumeByPriceModels: volumeByPriceModels,
	}
}
