package components

import (
	"QuoteService/proto"
	"time"

	logger "github.com/sirupsen/logrus"
	googleProto "google.golang.org/protobuf/proto"
)

var (
	quotesEventRkName = "rk.QuotesEvent"

	quotesEventMarshalErrMsg = "Error while marshal QuotesEvent"
	quoteProcessingErr       = "Error while processing update quotes: %s"

	publishedQuotesEventMsg         = "QuoteService published QuotesEvent: %+v"
	publishedScheduleQuotesEventMsg = "QuoteService published schedule QuotesEvent: %+v"
)

func (q *QuoteComponent) UpdateQuotes(byteMatchOrdersEvent []byte) {
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

	matchedOrder := matchOrdersEvent.LimitMatchedOrder
	currentQuotesEvent, err := q.Processing.UpdateQuotes(&matchedOrder.Pair, matchedOrder.InitPrice, matchOrdersEvent.MatchedVolume)
	if err != nil {
		logger.Debugf(quoteProcessingErr, err.Error())
		return
	}

	q.sendQuotesEvent(currentQuotesEvent)
	logger.Infof(publishedQuotesEventMsg, currentQuotesEvent.String())
}

func (q *QuoteComponent) SendCurrentQuotesEventBySchedule(sendQuotesEventScheduleTime time.Duration) {
	for {
		time.Sleep(sendQuotesEventScheduleTime)

		quotesEvent, err := q.Processing.GetQuotesEvent()
		if err != nil {
			logger.Errorf(quoteProcessingErr, err.Error())
			return
		}

		q.sendQuotesEvent(quotesEvent)
		logger.Infof(publishedScheduleQuotesEventMsg, quotesEvent.String())
	}
}

func (q *QuoteComponent) sendQuotesEvent(currentQuotesEvent *proto.QuotesEvent) {
	sendBody, err := googleProto.Marshal(currentQuotesEvent)
	if err != nil {
		logger.Errorf(quotesEventMarshalErrMsg, err.Error())
		return
	}

	q.RabbitProvider.SendMessage(quoteServiceExchangeName, quotesEventRkName, sendBody)
}
