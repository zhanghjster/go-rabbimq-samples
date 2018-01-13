### 介绍

RabbitMQ的Golang实例代码

### 环境搭建

环境为mac，执行 docker-compose -f docker-compose.yml up -d 启动rabbitmq，浏览器中打开http://localhost:15672 进入rabbitmq的管理界面，账号密码默认为 guest:guest

### 代码结构

1. main.go 为入口程序, 提供命令行解析和一些全局变量的初始化
2. customer.go 为消息消费端实例代码的定义，Customer 结构的方法为一些要运行的实例，注释中标注了运行这个实例的方式
3. producer.go 为消息发布端代码的定义，结构和使用方法同Customer
4. utils.go 一些工具函数

### 支持实例

#### DefaultExchange

测试default exchange的代码， 模拟queue默认绑定到default exchange， publisher通过直接使用queue的名name作为routing key将其发送给这个queue的情形

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

#### CompetingCustomer

测试多个customer竞争消费一个queue的情况，涉及customer处理message时间均等与悬殊的处理方式

1. 模拟customer具有相同效率，autoAck, 运行方式:

   * producer

     ~~~shell
     go run *.go -r producer -t CompetingCustomer \
           --message-body "hello world" --message-count 6 \
           -q hello
     ~~~

   * customer

     ~~~shell
     go run *.go -r customer -t CompetingCustomer  -q hello \
            --customer-count 3
     ~~~

   从日志可以看到，采用autoAck，message会被均分到每个customer


2. 模拟customer效率差别很大的情况，打开prefetch，关闭ack，运行方式:

   * producer

     ~~~shell
     go run *.go -r producer -t CompetingCustomer \
           --message-body "hello world" --message-count 6 \
           -q hello
     ~~~

     ​

   * customer

     ~~~shell
     go run *.go -r customer -t CompetingCustomer  -q hello \
          --customer-count 3 --customer-disparities
     ~~~

   日志显示，消费的比较快的customer会消费更多的message，而不是均分



