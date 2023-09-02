package main

import (
	"andriiklymiuk/go_aws_sqs_listener/utils"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"

	_ "github.com/lib/pq"
)

const Version = "1.0.0"

type MessageData struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Message   string    `json:"message"`
}

type QueueMessage struct {
	Data MessageData `json:"data"`
}

func pingDatabaseUntilConnected(db *sql.DB, maxRetries int, retryInterval time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		err := db.Ping()
		if err == nil {
			return nil
		}
		fmt.Printf(utils.RedColor, "Error pinging the database: %s. Retrying in %s...\n", err, retryInterval, utils.WhiteColor)
		time.Sleep(retryInterval)
	}
	return fmt.Errorf("exceeded maximum number of retries")
}

func main() {
	envConfig, err := utils.LoadConnectionConfig()
	if err != nil {
		log.Panicln("Couldn't load env variables", err)
	}

	dbConnection := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		envConfig.DatabaseHost,
		envConfig.DatabasePort,
		envConfig.DatabaseUser,
		envConfig.DatabasePassword,
		envConfig.DatabaseName,
	)

	db, err := sql.Open("postgres", dbConnection)
	if err != nil {
		fmt.Println("error during database connection", err)
	}

	defer db.Close()

	err = pingDatabaseUntilConnected(db, 10, 1*time.Second)
	if err != nil {
		fmt.Println(err, "Pinging postgres db failed: ", err.Error())
		return
	}

	fmt.Println(utils.GreenColor, "postgres db connection is ok", utils.WhiteColor)

	server := &http.Server{Addr: fmt.Sprintf(":%d", envConfig.ServerPort)}
	http.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})

	http.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		type versionInfo struct {
			Version string `json:"version"`
		}

		info := versionInfo{
			Version: Version,
		}

		jsonData, err := json.Marshal(info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	})

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped serving new connections.")
	}()

	var queueConnection = utils.SqsConnection{
		QueueUrl:             envConfig.AwsSqsQueueUrl,
		Region:               envConfig.AwsSqsRegion,
		MaxMessagesToProcess: 2,
		OnMessageReceived: func(rawMessage *sqs.Message, acknowledgeMessage func()) {
			var message QueueMessage
			err := json.Unmarshal([]byte(*rawMessage.Body), &message)
			if err != nil {
				fmt.Println(utils.RedColor, err, utils.WhiteColor)
				return
			}
			fmt.Println(
				"GO server received message: ",
				message.Data.Message,
				"with id",
				message.Data.Id,
			)
			acknowledgeMessage()
		},
	}
	queueConnection.EstablishConnection()

	gracefulShutdown := make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, syscall.SIGINT, os.Interrupt, syscall.SIGTERM)

	<-gracefulShutdown

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}

	fmt.Println("\n🚀 Bye, see you next time 🚀!")
}
