package handlers

import (
	"github.com/companieshouse/chs.go/log"
	"github.com/gorilla/pat"
)

func Init(r *pat.Router) {
	log.Info("initialising healthcheck endpoint beneath basePath: /refund-request-consumer")
	appRouter := r.PathPrefix("/refund-request-consumer").Subrouter()
	appRouter.Path("/healthcheck").Methods("GET").HandlerFunc(HealthCheck)
}
