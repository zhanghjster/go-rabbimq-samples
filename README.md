### 介绍

RabbitMQ的Golang实例代码

### 环境搭建

环境为mac，执行 docker-compose -f docker-compose.yml up -d 启动rabbitmq，浏览器中打开http://localhost:15672 进入rabbitmq的管理界面，账号密码默认为 guest:guest

### 代码结构

1. main.go 为入口程序, 提供命令行解析和一些全局变量的初始化
2. customer.go 为消息消费端实例代码的定义，Customer 结构体的***方法***为一些要运行的***实例***的名称，注释中标注了运行这个实例的方式
3. producer.go 为消息发布端代码的定义，结构和使用方法同Customer
4. utils.go 一些工具函数

### 实例列表

#### FanOutExchange

Fanout类型的exchange会将消息发送给所有绑定到它的queue，经典的广播模式。代码模拟的是publisher广播消息给所有的customer。

1. 运行producer

   ~~~shell
   go run *.go -r producer -t FanOutExchange  \
       --message-body "hello world" \
       --message-count 6 \
       --exchange fanoutSample
   ~~~

2. 运行customer

   ~~~shell
   go run *.go -r customer -t FanOutExchange  \
   	--customer-count 3 \
   	--exchange fanoutSample
   ~~~



#### DirectExchange

Direct 类型的Exchange根据routing key来发送消息给queue, 属于单播模式，工作原理如下：

一个queue使用 binding key **K** 绑定到Exchange，当一个消息使用 **R** 为routing key时，exchange会将其发送给 **k=R**的queue

这种类型的Exchange通常用于指定message的接收者的情景，实现过滤的概念，queue通过binding key来限制它感兴趣的消息。

代码模拟的是广播日志，一个queue只接收的是Warning类型的日志，另一个queue接收'Error'和'Fatal'类型的日志

1. 运行producer

   ~~~shell
   go run *.go -r producer -t DirectExchange \
   	--message-body "log here..." \
   	--message-count 10 \
   	--exchange directExchangeSample
   ~~~

   ​

2. 运行customer

   ~~~shell
   go run *.go -r customer -t DirectExchange \
   --exchange directExchangeSample
   ~~~

#### DefaultExchange

每一个queue都会自动的绑定到default exchange，binding key 用的是queue的name。代码模拟的是publisher通过直接使用queue的name作为routing key将消息直接发送给指定名称的queue。

运行方式：

1. 运行producer

   ```shell
   go run *.go -r producer -t DefaultExchange \
        --message-body "hello world" \
   	 -q hello
   ```

2. 运行customer

   ```shell
   go run *.go -r customer -t DefaultExchange  -q hello
   ```

#### Topic Exchange

Topic exchange 与 Direct exchange的使用方法类似，不同之处是它是根据routing key与binding key的做类似正则匹配来进行路由，而不是像direct change那样需要完全相等。比如, queue的binding key是 "#.error.#"会接收所有routing key中带有 ".error."的消息

实例模拟的是两个队列，一个关心所有app的error日志，另一个队列关心的是“chat"这个app的所有日志，消息的routing key的命名规则为"<app_name>.<error_level>".  第一个queue使用 "#.error"作为binding key，第二个queue用"chat.#"作为binding key

运行方式：

1. producer

   ~~~shell
   go run *.go -r producer -t TopicExchange \
    	--message-body "log here..." \
    	--message-count 10 \
    	--exchange topicExchangeSample
   ~~~

2. customer

   ~~~shell
   go run *.go -r customer -t TopicExchange --exchange topicExchangeSample
   ~~~

#### CompetingCustomer

代码模拟的是多个customer竞争消费一个queue的情况，涉及customer处理message时间均等与悬殊的处理方式

1. 模拟customer具有相同效率，均等的消费消息，autoAck为true, 运行方式:

   * producer

     ~~~shell
     go run *.go -r producer -t CompetingCustomer \
           --message-body "hello world" \
           --message-count 6 \
           -q hello
     ~~~

   * customer

     ~~~shell
     go run *.go -r customer -t CompetingCustomer \
            --customer-count 3 \
            -q hello
     ~~~


2. 模拟customer效率差别很大的情况，打开prefetch，关闭ack，运行方式:

   * producer

     ~~~shell
     go run *.go -r producer -t CompetingCustomer \
           --message-body "hello world" \
           --message-count 6 \
           -q hello
     ~~~

     ​

   * customer

     ~~~shell
     go run *.go -r customer -t CompetingCustomer \
          --customer-count 3 \
          --customer-disparities \
          -q hello
     ~~~

   ​

