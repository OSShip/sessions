package events

import (
	"context"

	"github.com/OSShip/utils/kafka"
)

type Publisher struct {
	producer *kafka.Producer
}

func New(brokers string) *Publisher {
	return &Publisher{producer: kafka.NewProducer(brokers, "session.events")}
}

func (p *Publisher) Close() {
	p.producer.Close()
}

func (p *Publisher) PublishScheduled(ctx context.Context, payload map[string]string) error {
	return p.producer.Publish(ctx, "session.scheduled", payload)
}

func (p *Publisher) PublishReminderDue(ctx context.Context, payload map[string]string) error {
	return p.producer.Publish(ctx, "session.reminder_due", payload)
}
