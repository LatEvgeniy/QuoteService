package main

import (
	"QuoteService/components"
	"QuoteService/processing"
	"QuoteService/providers"
	"QuoteService/sandbox"
	"time"
)

var (
	sendMarketDepthEventScheduleTime = 10 * time.Second
	sendQuotesEventScheduleTime      = 10 * time.Second

	orderProcessingExchangeName = "ex.OrderProcessingService"
	quoteServiceExchangeName    = "ex.QuoteService"

	createOrderResponseRkName = "rk.CreateOrderResponse"
	matchOrdersEventRkName    = "rk.MatchOrdersEvent"
	removeOrderResponseRkName = "rk.RemoveOrderResponse"

	createOrderResponseListenerQueueName         = "q.QuoteService.CreateOrderResponse.Listener"
	removeOrderResponseListenerQueueName         = "q.QuoteService.RemoveOrderResponse.Listener"
	marketDepthMatchOrdersEventListenerQueueName = "q.QuoteService.MarketDepth.MatchOrdersEvent.Listener"
	quotesMatchOrdersEventListenerQueueName      = "q.QuoteService.Quotes.MatchOrdersEvent.Listener"
)

func main() {
	rabbitProvider := providers.NewRabbitProvider()
	redisClient := providers.NewRedisClient()
	defer redisClient.Close()

	quoteProcessing := &processing.QuoteProcessing{RedisClient: redisClient}
	quoteComponent := &components.QuoteComponent{RabbitProvider: rabbitProvider, Processing: quoteProcessing}

	sandbox := &sandbox.Sandbox{RabbitProvider: rabbitProvider}

	rabbitProvider.DeclareExchange(quoteServiceExchangeName)

	go quoteComponent.SendMarketDepthEventBySchedule(sendMarketDepthEventScheduleTime)
	go quoteComponent.SendCurrentQuotesEventBySchedule(sendQuotesEventScheduleTime)

	go sandbox.RunSandbox()

	runListeners(rabbitProvider, quoteComponent)
}

func runListeners(rabbitProvider *providers.RabbitProvider, quoteComponent *components.QuoteComponent) {
	msgs, ch := rabbitProvider.GetQueueConsumer(orderProcessingExchangeName, createOrderResponseRkName, createOrderResponseListenerQueueName)
	go rabbitProvider.RunListener(msgs, ch, quoteComponent.UpdateMarketDepthByCreateOrderResponse)

	msgs, ch = rabbitProvider.GetQueueConsumer(orderProcessingExchangeName, removeOrderResponseRkName, removeOrderResponseListenerQueueName)
	go rabbitProvider.RunListener(msgs, ch, quoteComponent.UpdateMarketDepthByRemoveOrderResponse)

	msgs, ch = rabbitProvider.GetQueueConsumer(orderProcessingExchangeName, matchOrdersEventRkName, quotesMatchOrdersEventListenerQueueName)
	go rabbitProvider.RunListener(msgs, ch, quoteComponent.UpdateQuotes)

	msgs, ch = rabbitProvider.GetQueueConsumer(orderProcessingExchangeName, matchOrdersEventRkName, marketDepthMatchOrdersEventListenerQueueName)
	rabbitProvider.RunListener(msgs, ch, quoteComponent.UpdateMarketDepthByMatchOrdersEvent)
}
