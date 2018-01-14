package main

import (
	"strconv"

	"fmt"

	"math/rand"

	"github.com/streadway/amqp"
)

type Producer struct{}

// 消息会发送给每一个绑定到fanout类型的exchange的queue
// publish时的routing key会被忽略，经典的广播模式
//
// $go run *.go -r producer -t FanOutExchange  \
//  --message-body "hello world" \
//  --message-count 6 \
// 	--exchange fanoutSample
func (p *Producer) FanOutExchange() {
	// 定义一个fanout类型的exchange
	err := AmqpChan.ExchangeDeclare(
		ExchangeName,
		"fanout",
		false, false, false, false,
		nil,
	)
	FatalErr(err)

	for i := 1; i < Message.Count; i++ {
		msg := fmt.Sprintf("%d %s", i, Message.Body)
		err := AmqpChan.Publish(
			ExchangeName,
			"", // fanout类型的exchange忽略routing key
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

// Direct 类型的Exchange根据routing key来发送消息给queue, 属于单播模式
// 消息发送给binding key和消息的routing key相同的queue
// 通常用于指定message的接收者的情景, queue通过binding key来限制它感兴趣的消息。
//
// 代码模拟的是广播日志，
// 一个queue只接收的是Warning类型的日志，
// 另一个queue接收'Error'和'Fatal'类型的日志
//
// $ go run *.go -r producer -t DirectExchange \
// 	--message-body "log here..." \
// 	--message-count 10 \
// 	--exchange directExchangeSample
func (p *Producer) DirectExchange() {
	err := AmqpChan.ExchangeDeclare(
		ExchangeName,
		"direct",
		false, false, false, false,
		nil,
	)
	FatalErr(err)

	var levels = []string{"warning", "error", "fatal"}
	// 定义关注warning的queue
	q, err := AmqpChan.QueueDeclare("", false, false, false, false, nil)
	FatalErr(err)
	err = AmqpChan.QueueBind(q.Name, levels[0], ExchangeName, false, nil)
	FatalErr(err)

	// 定义关注error和fatal的queue
	q, err = AmqpChan.QueueDeclare("", false, false, false, false, nil)
	FatalErr(err)
	for _, l := range levels[1:] {
		err = AmqpChan.QueueBind(q.Name, l, ExchangeName, false, nil)
	}

	for i := 0; i < Message.Count; i++ {
		level := levels[rand.Intn(len(levels))]
		msg := fmt.Sprintf("[%s]%s", level, Message.Body)
		err := AmqpChan.Publish(
			ExchangeName,
			level,
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

// 模拟的是两个队列，一个关心所有app的error日志，另一个队列关心的是“chat"这个app的所有日志
// 消息的routing key的命名规则为"<app_name>.<error_level>"
//
// go run *.go -r producer -t TopicExchange \
// 	--message-body "log here..." \
// 	--message-count 10 \
// 	--exchange topicExchangeSample
func (p *Producer) TopicExchange() {
	err := AmqpChan.ExchangeDeclare(
		ExchangeName,
		"topic",
		false,
		true, // auto delete
		false,
		false,
		nil,
	)
	FatalErr(err)

	var levels = []string{"warning", "error", "fatal"}
	var apps = []string{"chat", "live", "image"}
	for _, key := range []string{"chat.#", "#.error"} {
		q, err := AmqpChan.QueueDeclare(QueueName, false, true, false, false, nil)
		FatalErr(err)

		err = AmqpChan.QueueBind(q.Name, key, ExchangeName, false, nil)
		FatalErr(err)
	}

	for i := 1; i < Message.Count; i++ {
		var key = fmt.Sprintf(
			"%s.%s", apps[rand.Intn(len(apps))], levels[rand.Intn(len(levels))],
		)

		AmqpChan.Publish(
			ExchangeName,
			key,
			false, false,
			amqp.Publishing{
				ContentType: "plain/text",
				Body:        []byte(key + " " + Message.Body),
			},
		)

		Log.Infof("send %s, message '%s'", key, Message.Body)
	}

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
