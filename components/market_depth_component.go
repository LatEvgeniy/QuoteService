package components

import (
	"QuoteService/processing"
	"QuoteService/proto"
	"QuoteService/providers"
	"time"

	logger "github.com/sirupsen/logrus"
	googleProto "google.golang.org/protobuf/proto"
)

type QuoteComponent struct {
	RabbitProvider *providers.RabbitProvider
	Processing     *processing.QuoteProcessing
}

var (
	quoteServiceExchangeName = "ex.QuoteService"
	marketDepthEventRkName   = "rk.MarketDepthEvent"

	gotMCreateOrderResponseMsg   = "QuoteService got CreateOrderResponse: %s\n"
	gotErrCreateOrderResponseMsg = "QuoteService got CreateOrderResponse with market order or with err: %s. Skipping\n"
	gotRemoveOrderResponseMsg    = "QuoteService got RemoveOrderResponse: %s\n"
	gotErrRemoveOrderResponseMsg = "QuoteService got RemoveOrderResponse with err: %s. Skipping\n"
	gotMatchOrdersEventMsg       = "QuoteService got MatchOrdersEvent: %s\n"
	gotErrMatchOrdersEventMsg    = "QuoteService got MatchOrdersEvent with err: %s. Skipping\n"

	noMarketDepthMsg = "No Market Depth, skipping send schedule MarketDepthEvent"

	unmarshalCreateOrderResponseErrMsg = "Error while unmarshal CreateOrderResponse"
	unmarshalRemoveOrderResponseErrMsg = "Error while unmarshal RemoveOrderResponse"
	unmarshalMatchOrdersEventErrMsg    = "Error while unmarshal MatchOrdersEvent"
	marketDepthEventMarshalErrMsg      = "Error while marshal MarketDepthEvent"
	marketDepthProcessingErr           = "Error while processing update marketDepth: %s"

	publishedMarketDepthEventMsg         = "QuoteService published MarketDepthEvent: %+v"
	publishedScheduleMarketDepthEventMsg = "QuoteService published schedule MarketDepthEvent: %+v"
)

func (q *QuoteComponent) UpdateMarketDepthByCreateOrderResponse(byteCreateOrderRresponse []byte) {
	var createOrderResponse proto.CreateOrderResponse
	if err := googleProto.Unmarshal(byteCreateOrderRresponse, &createOrderResponse); err != nil {
		logger.Error(unmarshalCreateOrderResponseErrMsg)
		return
	}

	if createOrderResponse.CreatedOrder.Type == proto.OrderType_MARKET || createOrderResponse.Error != nil {
		logger.Debugf(gotErrCreateOrderResponseMsg, createOrderResponse.String())
		return
	}

	logger.Infof(gotMCreateOrderResponseMsg, createOrderResponse.String())

	createdOrder := createOrderResponse.CreatedOrder
	marketDepthEvent, err := q.Processing.UpdateMarketDepth(createdOrder.Direction.String(), createdOrder.Pair.String(), createdOrder.InitPrice, createdOrder.InitVolume)
	if err != nil {
		logger.Errorf(marketDepthProcessingErr, err.Error())
		return
	}

	q.sendMarketDepthEventEvent(marketDepthEvent)
	logger.Infof(publishedMarketDepthEventMsg, marketDepthEvent.String())
}

func (q *QuoteComponent) UpdateMarketDepthByRemoveOrderResponse(byteRemoveOrderResponse []byte) {
	var removeOrderResponse proto.RemoveOrderResponse
	if err := googleProto.Unmarshal(byteRemoveOrderResponse, &removeOrderResponse); err != nil {
		logger.Error(unmarshalRemoveOrderResponseErrMsg)
		return
	}

	if removeOrderResponse.Error != nil {
		logger.Debugf(gotErrRemoveOrderResponseMsg, removeOrderResponse.String())
		return
	}

	logger.Infof(gotRemoveOrderResponseMsg, removeOrderResponse.String())

	removedOrder := removeOrderResponse.RemovedOrder
	updateVolume := -(removedOrder.InitVolume - removedOrder.FilledVolume)
	marketDepthEvent, err := q.Processing.UpdateMarketDepth(removedOrder.Direction.String(), removedOrder.Pair.String(), removedOrder.InitPrice, updateVolume)
	if err != nil {
		logger.Errorf(marketDepthProcessingErr, err.Error())
		return
	}

	q.sendMarketDepthEventEvent(marketDepthEvent)
	logger.Infof(publishedMarketDepthEventMsg, marketDepthEvent.String())
}

func (q *QuoteComponent) UpdateMarketDepthByMatchOrdersEvent(byteMatchOrdersEvent []byte) {
	var matchOrdersEvent proto.MatchOrdersEvent
	if err := googleProto.Unmarshal(byteMatchOrdersEvent, &matchOrdersEvent); err != nil {
		logger.Error(unmarshalMatchOrdersEventErrMsg)
		return
	}

	if matchOrdersEvent.Error != nil {
		logger.Debugf(gotErrMatchOrdersEventMsg, matchOrdersEvent.String())
		return
	}

	logger.Infof(gotMatchOrdersEventMsg, matchOrdersEvent.String())

	limitOrders := []proto.Order{*matchOrdersEvent.LimitMatchedOrder}
	if matchOrdersEvent.CreatedMatchedOrder.Type == proto.OrderType_LIMIT {
		limitOrders = append(limitOrders, *matchOrdersEvent.CreatedMatchedOrder)
	}

	var marketDepthEvent *proto.MarketDepthEvent
	var err error
	for _, limitOrder := range limitOrders {
		marketDepthEvent, err = q.Processing.UpdateMarketDepth(
			limitOrder.Direction.String(), limitOrder.Pair.String(), limitOrder.InitPrice, -matchOrdersEvent.MatchedVolume)
		if err != nil {
			logger.Errorf(marketDepthProcessingErr, err.Error())
			return
		}
	}

	q.sendMarketDepthEventEvent(marketDepthEvent)
	logger.Infof(publishedMarketDepthEventMsg, marketDepthEvent.String())
}

func (q *QuoteComponent) SendMarketDepthEventBySchedule(sendMarketDepthEventScheduleTime time.Duration) {
	for {
		time.Sleep(sendMarketDepthEventScheduleTime)

		marketDepthEvent, err := q.Processing.GetMarketDepthEvent()
		if err != nil {
			logger.Errorf(marketDepthProcessingErr, err.Error())
			return
		}

		if marketDepthEvent.MarketDepth == nil {
			logger.Debug(noMarketDepthMsg)
			continue
		}

		q.sendMarketDepthEventEvent(marketDepthEvent)
		logger.Infof(publishedScheduleMarketDepthEventMsg, marketDepthEvent.String())
	}
}

func (q *QuoteComponent) sendMarketDepthEventEvent(marketDepthEvent *proto.MarketDepthEvent) {
	sendBody, err := googleProto.Marshal(marketDepthEvent)
	if err != nil {
		logger.Errorf(marketDepthEventMarshalErrMsg, err.Error())
		return
	}

	q.RabbitProvider.SendMessage(quoteServiceExchangeName, marketDepthEventRkName, sendBody)
}
