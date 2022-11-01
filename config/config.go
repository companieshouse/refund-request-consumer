package config

import (
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/gofigure"
)

// Config is the filing processed tx updater config.
type Config struct {
	gofigure               interface{} `order:"env,flag"`
	BrokerAddr             []string    `env:"KAFKA_BROKER_ADDR"                 flag:"broker-addr"                         flagDesc:"Kafka broker address"`
	SchemaRegistryURL      string      `env:"SCHEMA_REGISTRY_URL"               flag:"schema-registry-url"                 flagDesc:"Schema registry url"`
	ZookeeperChroot        string      `env:"KAFKA_ZOOKEEPER_CHROOT"            flag:"zookeeper-chroot"                    flagDesc:"Zookeeper chroot"`
	ZookeeperURL           string      `env:"KAFKA_ZOOKEEPER_ADDR"              flag:"zookeeper-addr"                      flagDesc:"Zookeeper address"`
	ConsumerGroupName      string      `env:"REFUND_REQUEST_GROUP_NAME"         flag:"refund-request-group-name"                 flagDesc:"Refund Request Group Name"`
	ConsumerRetryGroupName string      `env:"REFUND_REQUEST_RETRY_GROUP_NAME"   flag:"refund-request-retry-group-name"           flagDesc:"Refund Request retry Group Name"`
	ConsumerTopic          string      `env:"REFUND_REQUEST_TOPIC"              flag:"refund-request-topic"              flagDesc:"Refund Request topic"`
	ConsumerTopicOffset    int64       `env:"REFUND_REQUEST_TOPIC_OFFSET"       flag:"refund-request-topic-offset"       flagDesc:"Refund Request topic offset value"`
	RetryTopicOffset       int64       `env:"REFUND_REQUEST_RETRY_TOPIC_OFFSET" flag:"refund-request-retry-topic-offset" flagDesc:"Refund Request retry topic offset value"`
	RetryThrottleRate      int         `env:"RETRY_THROTTLE_RATE_SECONDS"       flag:"retry-throttle-rate-seconds"         flagDesc:"Retry throttle rate seconds"`
	MaxRetryAttempts       int         `env:"MAXIMUM_RETRY_ATTEMPTS"            flag:"max-retry-attempts"                   flagDesc:"Maximum retry attempts"`
	IsErrorConsumer        bool        `env:"IS_ERROR_QUEUE_CONSUMER"           flag:"is-error-queue-consumer"             flagDesc:"Set this flag if it is an error queue consumer"`
	// PaymentsAPIURL         string      `env:"PAYMENTS_API_URL"                  flag:"payments-api-url"                                               flagDesc:"Base URL for the Payment Service API"`
	// ChsAPIKey              string      `env:"CHS_API_KEY"                       flag:"chs-api-key"                         flagDesc:"API access key"`
}

// Namespace implements service.Config.Namespace.
func (c *Config) Namespace() string {
	return "refund-request-consumer"
}

var cfg *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		//ZookeeperURL:                    "",
		ZookeeperChroot:   "",
		ConsumerGroupName: "refund-request-consumer",
		//ConsumerRetryGroupName: "refund-request-consumer-retry",
		ConsumerTopic:       "refund-request",
		ConsumerTopicOffset: int64(-1),
		//FilingProcessedRetryTopicOffset: int64(-1),
		RetryThrottleRate: 3,
		MaxRetryAttempts:  2,
	}

	err := gofigure.Gofigure(cfg)
	if err != nil {
		log.Error(err, nil)

		return nil, err
	}

	return cfg, nil
}
