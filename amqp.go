package logger

import (
	"context"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

const amqpDeliveryName = "x-log-process-id"

func FromAMQP(ctx context.Context, e amqp.Delivery, namespace string) (context.Context, *zap.Logger) {
	if ctx == nil {
		ctx = context.Background()
	}

	processID, ok := e.Headers[amqpDeliveryName].(string)
	if !ok {
		processID = uuid.NewString()
	}

	BindFields(ctx, MQMessageID(e.MessageId))

	ctx = BindProcessID(ctx, processID)

	return ctx, FromCtx(ctx, namespace)
}

func ToAMQPHeader(ctx context.Context, table amqp.Table) amqp.Table {
	if table == nil {
		table = make(map[string]interface{})
	}
	table[amqpDeliveryName] = GetProcessID(ctx)

	return table
}
