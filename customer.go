package main

import (
	"strconv"

	"math/rand"

	"time"

	"github.com/streadway/amqp"
)

type Customer struct{}

// 消息会发送给每一个绑定到fanout类型的exchange的queue
// publish时的routing key会被忽略，经典的广播模式
//
// go run *.go -r customer -t FanOutExchange  \
// 	--customer-count 3 \
// 	--exchange fanoutSample
func (c *Customer) FanOutExchange() {
	// 定义一个fanout类型的exchange
	err := AmqpChan.ExchangeDeclare(
		ExchangeName,
		"fanout",
		false, false, false, false,
		nil,
	)
	FatalErr(err)

	// 每个customer一个queue
	for i := 1; i <= CustomerCount; i++ {
		q, err := AmqpChan.QueueDeclare(
			"", // 系统默认创建一个
			false, false, false, false,
			nil,
		)
		FatalErr(err)

		// 绑定到fanoutSample这个Exchange
		err = AmqpChan.QueueBind(
			q.Name,
			"", // fanout类型exchange忽略binding key
			ExchangeName,
			false,
			nil,
		)
		FatalErr(err)

		delivery, err := AmqpChan.Consume(
			q.Name,
			"customer."+strconv.Itoa(i),
			true, // autoAck
			false,
			false,
			false,
			nil,
		)
		go func(i int) {
			Log.Infof("customer %d start consume queue %s", i, q.Name)
			for {
				msg, _ := <-delivery
				Log.Infof("customer %d get message '%s'", i, string(msg.Body))
			}
		}(i)
	}

	WaitForTERM()
}

// Direct 类型的Exchange根据routing key来发送消息给queue, 属于单播模式
// 消息发送给binding key和消息的routing key相同的queue
// 通常用于指定message的接收者的情景, queue通过binding key来限制它感兴趣的消息。
//
// 代码模拟的是广播日志，
// 一个queue只接收的是Warning类型的日志，
// 另一个queue接收'Error'和'Fatal'类型的日志
//
// $ go run *.go -r customer -t DirectExchange \
// 	--exchange directExchangeSample
func (c *Customer) DirectExchange() {
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

	delivery, err := AmqpChan.Consume(
		q.Name, "", false, false, false, false, nil,
	)
	FatalErr(err)
	go c.dumpMessage("queue for warning", delivery)

	// 定义关注error和fatal的queue
	q, err = AmqpChan.QueueDeclare("", false, false, false, false, nil)
	FatalErr(err)
	for _, l := range levels[1:] {
		err = AmqpChan.QueueBind(q.Name, l, ExchangeName, false, nil)
	}
	delivery, err = AmqpChan.Consume(
		q.Name, "", false, false, false, false, nil,
	)
	FatalErr(err)
	go c.dumpMessage("queue for error and fatal", delivery)

	WaitForTERM()
}

// 新的Queue会自动的以它的name为binding key
// 绑定到default exchange
//
// producer在publish message时不指定exchanger，
// 用queue的名称作为routing key则发送到指定名称的队列
//
// go run *.go -r customer -t DefaultExchange  -q hello
func (c *Customer) DefaultExchange() {
	// 创建一个queue
	q, err := AmqpChan.QueueDeclare(
		QueueName,
		false, false, false, false,
		nil,
	)
	FatalErr(err)

	delivery, err := AmqpChan.Consume(
		q.Name, // 消费的队列名称
		"",     // 消费者名称
		true,   // 自动确认
		false,
		false,
		false,
		nil,
	)
	FatalErr(err)

	c.dumpMessage(q.Name, delivery)
}

// 模拟的是两个队列，一个关心所有app的error日志，另一个队列关心的是“chat"这个app的所有日志
// 消息的routing key的命名规则为"<app_name>.<error_level>"
//
// $ go run *.go -r customer -t TopicExchange \
// 	--exchange topicExchangeSample
func (c *Customer) TopicExchange() {
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

	for _, key := range []string{"chat.#", "#.error"} {
		q, err := AmqpChan.QueueDeclare(
			QueueName,
			false,
			true, // queue使用完后自动删除
			false,
			false,
			nil,
		)
		FatalErr(err)

		AmqpChan.QueueBind(q.Name, key, ExchangeName, false, nil)
		delivery, _ := AmqpChan.Consume(q.Name, "", true, false, false, false, nil)
		go c.dumpMessage("["+key+"]", delivery)
	}

	WaitForTERM()
}

// 根据header做更复杂的过滤规则
// $go run *.go -r producer -t HeaderExchange \
// 	--message-body "log here..." \
// 	--message-count 10 \
// 	--exchange headerExchangeSample

func (c *Customer) HeaderExchange() {
	err := AmqpChan.ExchangeDeclare(
		ExchangeName,
		"headers", // headers 类型
		false,
		true, // auto delete
		false,
		false,
		nil,
	)
	FatalErr(err)

	var args = []amqp.Table{
		{
			"app": "chat", "version": "latest", "x-match": "any",
		},
		{
			"app": "live", "version": "2.0", "x-match": "all",
		},
	}
	for _, arg := range args {
		q, err := AmqpChan.QueueDeclare(
			"", false, true, false, false, nil,
		)
		FatalErr(err)
		AmqpChan.QueueBind(q.Name, "", ExchangeName, false, arg)
		delivery, _ := AmqpChan.Consume(
			q.Name, "", true, false, false, false, nil,
		)

		go func(ag amqp.Table) {
			for {
				msg, _ := <-delivery
				Log.Infof("[%v] get message '%s' with header %v",
					ag, string(msg.Body), msg.Headers)
			}
		}(arg)
	}

	WaitForTERM()
}

// 多个消费者竞争消费同一个queue
//
// 1. 对于customer消费每一个message时间均等,
//    使用autoAck，消息会被customer均分
//    go run *.go -r customer -t CompetingCustomer  -q hello \
//       --customer-count 3
//
// 2. 对于customer消费每一个message时间差别很大
//    使用prefetch + 手动ack，消息会自动的被消费的比较快的customer消费
//    go run *.go -r customer -t CompetingCustomer  -q hello \
//     --customer-count 3 --customer-disparities
func (c *Customer) CompetingCustomer() {
	// 创建queue
	// 不对queue进行和exchange绑定设置
	// 则默认绑定到default exchange
	q, err := AmqpChan.QueueDeclare(
		QueueName,
		false, false, false, false,
		nil,
	)
	FatalErr(err)

	if CustomerDisparities {
		AmqpChan.Qos(
			1, // 设置可以预抓取的message个数
			0,
			false,
		)
	}

	// 如果消费者效率差别很大，关闭autoAck打开prefetch
	var autoAck = !CustomerDisparities

	for i := 1; i <= CustomerCount; i++ {
		// 模拟customer消费一个message的时间差别很大
		// 打开prefetch, customer那里还需要关闭autoAck

		// 一个新的customer开始消费上面创建的queue
		delivery, err := AmqpChan.Consume(
			q.Name,
			"customer"+strconv.Itoa(i),
			autoAck, // 自动ack，消息被customer取出后会被broker删除
			false, false, false,
			nil,
		)
		FatalErr(err)

		go func(i int) {
			Log.Infof("customer %d start consume queue %s, same effectiveness %v", i, q.Name, !CustomerDisparities)
			for {
				msg, _ := <-delivery

				Log.Infof("customer %d, get message %s", i, string(msg.Body))

				// 使用sleep模拟customer消费一个message处理时间悬殊情况
				if CustomerDisparities {
					time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
				}

				// 如果不autoAck，则需要手动
				if !autoAck {
					msg.Ack(false)
				}

				Log.Infof("customer %d process message done", i)
			}
		}(i)
	}

	WaitForTERM()
}

func (c *Customer) dumpMessage(preFix string, d <-chan amqp.Delivery) {
	for {
		msg, _ := <-d
		Log.Infof("%s get message '%s'", preFix, string(msg.Body))
	}
}
