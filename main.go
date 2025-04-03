//coverage:ignore file
package main

import (
	"fmt"
	goLog "log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/companieshouse/chs.go/kafka/resilience"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/refund-request-consumer/config"
	"github.com/companieshouse/refund-request-consumer/service"
)

func main() {
	log.Namespace = "refund-request-consumer"

	// Push the Sarama logs into our custom writer
	sarama.Logger = goLog.New(&log.Writer{}, "[Sarama] ", goLog.LstdFlags)

	cfg, err := config.Get()
	if err != nil {
		log.Error(fmt.Errorf("error configuring service: %w. Exiting", err), nil)
		return
	}

	log.Info("initialising refund-request-consumer service...")

	mainChannel := make(chan os.Signal, 1)
	retryChannel := make(chan os.Signal, 1)

	svc, err := service.New(cfg.ConsumerTopic, cfg.ConsumerGroupName, cfg.ConsumerTopicOffset, cfg, nil)
	if err != nil {
		log.Error(fmt.Errorf("error initialising main consumer service: '%w'. Exiting", err), nil)
		return
	}

	var wg sync.WaitGroup
	if !cfg.IsErrorConsumer {
		retrySvc, err := getRetryService(cfg)
		if err != nil {
			log.Error(fmt.Errorf("error initialising retry consumer service: '%w'. Exiting", err), nil)
			svc.Shutdown()
			return
		}
		wg.Add(1)
		go retrySvc.Start(&wg, retryChannel)
	}

	wg.Add(1)
	go svc.Start(&wg, mainChannel)

	waitForServiceClose(&wg, mainChannel, retryChannel)

	log.Info("Application successfully shutdown")

}

func getRetryService(cfg *config.Config) (*service.Service, error) {
	retry := &resilience.ServiceRetry{
		ThrottleRate: time.Duration(cfg.RetryThrottleRate),
		MaxRetries:   cfg.MaxRetryAttempts,
	}

	retrySvc, err := service.New(cfg.ConsumerTopic, cfg.ConsumerGroupName, cfg.RetryTopicOffset, cfg, retry)
	if err != nil {
		return nil, fmt.Errorf("error initialising retry consumer service: %w", err)
	}

	return retrySvc, nil
}

// waitForServiceClose will receive the close signal and forward a notification
// to all services (go routines) to ensure that they clean up (for example their
// consumers and producers) and exit gracefully.
func waitForServiceClose(wg *sync.WaitGroup, mainChannel, retryChannel chan os.Signal) {

	// Channel to fan-out interrupt/kill notifications
	notificationChannel := make(chan os.Signal, 1)
	signal.Notify(notificationChannel, os.Interrupt, os.Kill, syscall.SIGTERM)

	select {
	case notification := <-notificationChannel:
		// Falls into this block to successfully close consumer after service shutdown
		log.Info("Close signal received, fanning out...")
		log.Debug("Sending notification to main consumer channel")
		mainChannel <- notification

		log.Debug("Sending notification to retry consumer channel")
		retryChannel <- notification

		log.Info("Fan out completed")
	}
	wg.Wait()
}
