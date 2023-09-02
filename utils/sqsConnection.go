package utils

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SqsConnection struct {
	Connection *sqs.SQS
	QueueUrl   string
	Region     string

	MessagesChannel      chan *sqs.Message
	MaxMessagesToProcess int

	OnMessageReceived func(*sqs.Message, func())
}

func (c *SqsConnection) PollMessages() {
	for {
		output, err := c.Connection.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.QueueUrl),
			MaxNumberOfMessages: aws.Int64(4),
			WaitTimeSeconds:     aws.Int64(10),
		})

		if err != nil {
			fmt.Printf("failed to fetch sqs message %s\n", err)
		} else {
			fmt.Println(BlueColor, "waiting for sqs messages", WhiteColor)
		}

		for _, message := range output.Messages {
			fmt.Println("New message received from sqs", message.Body)
			c.MessagesChannel <- message
		}
	}
}

func (c *SqsConnection) DeleteMessage(msg *sqs.Message) {
	c.Connection.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.QueueUrl),
		ReceiptHandle: msg.ReceiptHandle,
	})
}

func (c *SqsConnection) EstablishConnection() {
	connection := session.Must(
		session.NewSessionWithOptions(
			session.Options{
				Config: aws.Config{
					Region:   aws.String(c.Region),
					Endpoint: aws.String(c.QueueUrl),
				},
			},
		),
	)

	c.Connection = sqs.New(connection)

	c.MessagesChannel = make(chan *sqs.Message, 2)

	maxMessagesGuardChannel := make(chan struct{}, c.MaxMessagesToProcess)
	go c.PollMessages()

	for message := range c.MessagesChannel {
		maxMessagesGuardChannel <- struct{}{}
		c.DeleteMessage(message)

		go c.OnMessageReceived(message, func() {
			c.DeleteMessage(message)
			<-maxMessagesGuardChannel
			fmt.Println(
				GreenColor,
				"Message is processed and acknowledged",
				WhiteColor,
			)
		})
	}

	fmt.Println(
		BlueColor,
		"Connection to sqs is established, listening to messages...",
		WhiteColor,
	)
}
