package utils

import (
	"fmt"

	"github.com/caarlos0/env/v7"
	"github.com/joho/godotenv"
)

type EnvConfig struct {
	ServerPort int `env:"PORT,required"`

	AwsSqsRegion   string `env:"REGION,required"`
	AwsSqsQueueUrl string `env:"AWS_SQS_QUEUE_URL,required"`

	// not used directly in the app, but used in aws sqs
	AwsAccessKeyId     string `env:"AWS_ACCESS_KEY_ID,required"`
	AwsSecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY,required"`

	// not used directly in the app, used in syncs for db manipulations
	DatabaseHost     string `env:"DB_HOST,required"`
	DatabaseUser     string `env:"DB_USER,required"`
	DatabaseName     string `env:"DB_NAME,required"`
	DatabasePort     int    `env:"DB_PORT,required"`
	DatabasePassword string `env:"DB_PASSWORD,required"`
}

func LoadConnectionConfig() (*EnvConfig, error) {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(fmt.Errorf("unable to load .env file: %e", err))
	}

	config := EnvConfig{}

	err = env.Parse(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to parse environment variables: %e", err)
	}

	return &config, nil
}
