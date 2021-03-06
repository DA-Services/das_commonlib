package kafka

import (
	"context"
	"github.com/Shopify/sarama"
	"log"
)

/**
 * Copyright (C), 2019-2020
 * FileName: consumer
 * Author:   LinGuanHong
 * Date:     2020/12/15 10:45 上午
 * Description:
 */

type KafkaMessageQueueConsumer struct {
	stop          bool
	ConsumerGroup sarama.ConsumerGroup
	Ctx           context.Context
}

func NewDefaultKafkaMessageQueueConsumer(brokers []string, groupName string, ctx context.Context) (*KafkaMessageQueueConsumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Fetch.Default = 1024
	config.Consumer.Fetch.Max = 1024 * 2
	client, err := sarama.NewConsumerGroup(brokers, groupName, config)
	if err != nil {
		return nil, err
	}
	return &KafkaMessageQueueConsumer{
		ConsumerGroup: client,
		Ctx:           ctx,
	}, nil
}

// handleMsg return true means will commit this msg after handle it
func (c *KafkaMessageQueueConsumer) ConsumeWithHandCommit(topics []string, handleMsg func(msg *sarama.ConsumerMessage) bool) {
	go func() {
		for {
			if c.Ctx.Err() != nil {
				return
			}
			if err := c.ConsumerGroup.Consume(c.Ctx, topics, &HandCommitConsumer{HandleMsg: handleMsg}); err != nil && !c.stop {
				log.Panicf("Error from consumer: %s", err.Error())
			}
		}
	}()
}

func (c *KafkaMessageQueueConsumer) Close() error {
	c.stop = true
	return c.ConsumerGroup.Close()
}

type HandCommitConsumer struct {
	HandleMsg func(msg *sarama.ConsumerMessage) bool // true,  msg means consume
}

func (consumer *HandCommitConsumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *HandCommitConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *HandCommitConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		// fmt.Println("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
		select {
		case <-session.Context().Done():
			return nil
		default:
			if consumer.HandleMsg != nil {
				if consumer.HandleMsg(message) {
					session.MarkMessage(message, "")
				}
			}
		}
	}
	return nil
}
