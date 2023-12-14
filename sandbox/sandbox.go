package sandbox

import (
	"QuoteService/converters"
	"QuoteService/proto"
	"QuoteService/providers"

	logger "github.com/sirupsen/logrus"
	googleProto "google.golang.org/protobuf/proto"
)

type Sandbox struct {
	RabbitProvider *providers.RabbitProvider
}

var (
	quoteServiceExchangeName = "ex.QuoteService"

	marketDepthEventRkName = "rk.MarketDepthEvent"
	quotesEventRkName      = "rk.QuotesEvent"

	MarketDepthEventListenerQueueName = "q.QuoteService.MarketDepthEvent.Listener"
	quotesEventListenerQueueName      = "q.QuoteService.QuotesEvent.Listener"

	unmarshalMarketDepthEventErrMsg = "Error while unmarshal MarketDepthEvent"
	unmarshalQuotesEventErrMsg      = "Error while unmarshal QuotesEvent"

	marketDepthEventContentMsg = "For pair: %s and direction: %s market depth is: %+v"
	quotesEventContentMsg      = "For pair: %s last match was with price: %f and volume: %f"
)

func (s *Sandbox) RunSandbox() {
	msgs, ch := s.RabbitProvider.GetQueueConsumer(quoteServiceExchangeName, marketDepthEventRkName, MarketDepthEventListenerQueueName)
	go s.RabbitProvider.RunListener(msgs, ch, s.processMarkerDepthEvent)

	msgs, ch = s.RabbitProvider.GetQueueConsumer(quoteServiceExchangeName, quotesEventRkName, quotesEventListenerQueueName)
	s.RabbitProvider.RunListener(msgs, ch, s.processQuoteEvent)
}

func (s *Sandbox) processMarkerDepthEvent(bytesMarketDepthEvent []byte) {
	var marketDepthEvent proto.MarketDepthEvent
	if err := googleProto.Unmarshal(bytesMarketDepthEvent, &marketDepthEvent); err != nil {
		logger.Error(unmarshalMarketDepthEventErrMsg)
		return
	}

	for _, pairMarketDepth := range marketDepthEvent.MarketDepth {
		pairMarketDepthModel := converters.ConvertPairMarketDepthToModel(pairMarketDepth)
		logger.Debugf(marketDepthEventContentMsg, pairMarketDepthModel.OrderPair, pairMarketDepthModel.OrderDirection, pairMarketDepthModel.VolumeByPriceModels)
	}
}

func (s *Sandbox) processQuoteEvent(bytesQuoteEvent []byte) {
	var quotesEvent proto.QuotesEvent
	if err := googleProto.Unmarshal(bytesQuoteEvent, &quotesEvent); err != nil {
		logger.Error(unmarshalQuotesEventErrMsg)
		return
	}

	for _, pairQuote := range quotesEvent.CurrentQuotes {
		logger.Debugf(quotesEventContentMsg, pairQuote.Pair.String(), pairQuote.Price, pairQuote.Volume)
	}
}
