package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/streadway/amqp"
)

var (
	Log *logrus.Logger

	RootCmd cobra.Command

	AmqpConn *amqp.Connection
	AmqpChan *amqp.Channel

	test    string
	amqpUrl string
	role    string
	list    bool

	QueueName           string
	ExchangeName        string
	CustomerCount       int
	CustomerDisparities bool
	Message             struct {
		Body  string
		Count int
	}
)

func init() {
	Log = logrus.New()

	RootCmd = cobra.Command{
		Use: "rb",
		Run: rootCmdRun,
	}

	RootCmd.Flags().StringVarP(&test, "test", "t", "", "test name")
	RootCmd.Flags().StringVar(&amqpUrl, "amqp", "amqp://guest:guest@localhost:5672", "amqp url")
	RootCmd.Flags().StringVarP(&role, "role", "r", "", "[customer|producer] role")
	RootCmd.Flags().BoolVarP(&list, "list", "l", false, "show test case list")
	RootCmd.Flags().StringVarP(&QueueName, "queue", "q", "", "queue name")
	RootCmd.Flags().StringVarP(&ExchangeName, "exchange", "e", "", "exchange name")
	RootCmd.Flags().StringVar(&Message.Body, "message-body", "", "message content")
	RootCmd.Flags().IntVar(&Message.Count, "message-count", 1, "message count")
	RootCmd.Flags().IntVar(&CustomerCount, "customer-count", 1, "consumer count")
	RootCmd.Flags().BoolVar(&CustomerDisparities, "customer-disparities", false, "customers' speed have big different")
}

func rootCmdRun(cmd *cobra.Command, args []string) {
	var err error
	AmqpConn, err = amqp.Dial(amqpUrl)
	FatalErr(err)

	AmqpChan, err = AmqpConn.Channel()
	FatalErr(err)

	var p interface{}
	switch role {
	case "customer":
		p = new(Customer)
	case "producer":
		p = new(Producer)
	default:
		cmd.Usage()
		os.Exit(1)
	}

	// show supported test list
	if list {
		fmt.Printf("Methods of %s :\n", role)

		tp := reflect.TypeOf(p)
		for i := 0; i < tp.NumMethod(); i++ {
			m := tp.Method(i)
			fmt.Printf("\t %s\n", m.Name)
		}

		os.Exit(0)
	}

	if test == "" || amqpUrl == "" {
		cmd.Usage()
		os.Exit(0)
	}

	m := reflect.ValueOf(p).MethodByName(test)
	if m.IsValid() {
		Log.Infof("test '%s', role '%s'", test, role)
		m.Call([]reflect.Value{})
	} else {
		Log.Infof("%s of %s not defined", test, role)
	}
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err.Error())
	}
}
