package models

type PairMarketDepthModel struct {
	OrderPair           string
	OrderDirection      string
	VolumeByPriceModels []VolumeByPriceModel
}

type VolumeByPriceModel struct {
	Price  float64
	Volume float64
}

type PairQuoteModel struct {
	OrderPair string
	Price     float64
	Volume    float64
}
