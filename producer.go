package main

import (
	"strconv"

	"github.com/streadway/amqp"
)

type Producer struct{}

// 测试Default exchange
//
// 新的Queue会自动的以它的name为binding key
// 绑定到default exchange
//
// producer在publish message时不指定exchanger，
// 用queue的名称作为routing key则发送到指定名称的队列
//
// go run *.go -r producer -t DefaultExchange \
// 	--message-body "hello world" \
//  -q hello
func (p *Producer) DefaultExchange() error {
	q, err := AmqpChan.QueueDeclare(
		QueueName,
		false, false, false, false,
		nil,
	)
	FatalErr(err)

	err = AmqpChan.Publish(
		"",     // exchange 名称为空则发给default exchange
		q.Name, // routing key 为queue name
		false,
		false,
		amqp.Publishing{
			ContentType: "plain/text",
			Body:        []byte(Message.Body),
		},
	)
	FatalErr(err)

	Log.Infof("sent message '%s'", Message.Body)

	return nil
}

// 向指定名称queue发送多个消息供多个consumer消费
// 1. 对于每个customer消费任务时间均等的情况下，
//    采用自动ack，message会均分到每个customer
//    go run *.go -r customer -t CompetingCustomer  -q hello \
//     --customer-count 3 --customer-disparities
// 2. 对于customer消费一个任务的时间不均等情况下，
//    采用prefetch+手动ack，让运行的快的consumer消费更多，避免等待
//    go run *.go -r producer -t CompetingCustomer \
//      --message-body "hello world" --message-count 6 \
//      -q hello
func (p *Producer) CompetingCustomer() {
	q, err := AmqpChan.QueueDeclare(
		QueueName,
		false, false, false, false,
		nil,
	)
	FatalErr(err)

	for i := 1; i <= Message.Count; i++ {
		var msg = strconv.Itoa(i) + " " + Message.Body
		err := AmqpChan.Publish(
			"",     // 发送给default exchange
			q.Name, // 以q.Name为routing key
			false, false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(msg),
			},
		)
		FatalErr(err)

		Log.Infof("send message '%s'", msg)
	}
}
