package main

import (
	"fmt"
	"github.com/streadway/amqp"
)

func main(){

	////here need to add code to init mq connection
	conn, err := amqp.Dial("amqp://admin:admin@10.122.46.24:5672/devops")
	defer conn.Close()
	if err != nil {
		fmt.Printf("connect to mq failed: %v",err)
	}


	ch, err := conn.Channel()
	defer ch.Close()

	if err !=nil {
		fmt.Printf("generate message queue channel failed: %v",err)
	}

	sync_message:="hello"

	err = ch.Publish("deployment-status-sync", "mq-test", false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(sync_message),
	})

	if err !=nil {
		fmt.Printf("generate message queue channel failed: %v",err)
	}

}
