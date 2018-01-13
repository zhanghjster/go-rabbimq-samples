package main

import (
	"fmt"
	"strconv"
	"sync"

	"math/rand"

	"time"

	"github.com/pkg/errors"
)

type Customer struct{}

// 测试Default exchange
//
// 新的Queue会自动的以它的name为binding key
// 绑定到default exchange
//
// producer在publish message时不指定exchanger，
// 用queue的名称作为routing key则发送到指定名称的队列
//
// go run *.go -r customer -t DefaultExchange  -q hello
func (c *Customer) DefaultExchange() error {
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

	for {
		msg, _ := <-delivery
		Log.Infof("get msg '%s'", string(msg.Body))
	}

	return nil
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

	for i := 1; i <= CustomerCount; i++ {
		// 打开一个channel代表一个customer
		// 在rabbitmq的channels里可以看到多个channel
		ch, err := AmqpConn.Channel()
		FatalErr(err)

		// 模拟customer消费一个message的时间差别很大
		// 打开prefetch, customer那里还需要关闭autoAck
		if CustomerDisparities {
			ch.Qos(
				1, // 设置可以预抓取的message个数
				0,
				false,
			)
		}

		// 如果消费者效率差别很大，关闭autoAck打开prefetch
		var autoAck = !CustomerDisparities

		// 消费上面创建的queue
		delivery, err := ch.Consume(
			q.Name,
			"customer"+strconv.Itoa(i),
			autoAck, // 自动ack，消息被customer取出后会被broker删除
			false, false, false,
			nil,
		)

		go func(i int) {
			Log.Infof("customer %d start consume queue %s, same effectiveness %v", i, q.Name, !CustomerDisparities)
			for {
				msg, ok := <-delivery
				if !ok {
					FatalErr(errors.New(fmt.Sprintf("customer %d can read delivery fail", i)))
				}

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

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
