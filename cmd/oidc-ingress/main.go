package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/go-chi/valve"
	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pwillie/oidc-ingress/pkg/handlers"
)

const (
	source = "oidc-ingress"
)

var (
	clientConfigs   string
	listenAddress   string
	internalAddress string
	versionFlag     bool
)

func init() {
	flag.StringVar(&clientConfigs, "clients", "", "OIDC clients config expressed in yaml")
	flag.StringVar(&listenAddress, "listen", ":8000", "Listen address")
	flag.StringVar(&internalAddress, "internal", ":9000", "Internal listen address")
	flag.BoolVar(&versionFlag, "version", false, "Version")
}

func main() {
	flag.Parse()

	PrintVersion()
	if versionFlag {
		return
	}

	// Setup the logger backend using sirupsen/logrus and configure
	// it to use a custom JSONFormatter. See the logrus docs for how to
	// configure the backend at github.com/sirupsen/logrus
	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{
		// disable, as we set our own
		DisableTimestamp: true,
	}

	// Our graceful valve shut-off package to manage code preemption and
	// shutdown signaling.
	valv := valve.New()
	baseCtx := valv.Context()

	// HTTP service running in this program as well. The valve context is set
	// as a base context on the server listener at the point where we instantiate
	// the server - look lower.
	r := handlers.NewRouter(logger)

	oidc, err := handlers.NewOidcHandler(clientConfigs)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialise OIDC handler")
	}
	r.Get("/auth/verify/{clientid}", oidc.VerifyHandler)
	r.Get("/auth/signin/{clientid}", oidc.SigninHandler)
	r.Get("/auth/callback/{clientid}", oidc.CallbackHandler)

	logger.Infof("Starting server at: %s", listenAddress)
	srv := http.Server{Addr: listenAddress, Handler: chi.ServerBaseContext(baseCtx, r)}

	i := chi.NewRouter()
	i.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		render.NoContent(w, r)
	})
	i.Get("/metrics", promhttp.Handler().ServeHTTP)

	logger.Infof("Starting monitoring server at: %s", internalAddress)
	mon := http.Server{Addr: internalAddress, Handler: chi.ServerBaseContext(baseCtx, i)}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			// sig is a ^C, handle it
			fmt.Println("shutting down..")

			// first valv
			valv.Shutdown(20 * time.Second)

			// create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			// start http shutdown
			srv.Shutdown(ctx)
			mon.Shutdown(ctx)

			// verify, in worst case call cancel via defer
			select {
			case <-time.After(21 * time.Second):
				fmt.Println("not all connections done")
			case <-ctx.Done():

			}
		}
	}()
	go func() {
		mon.ListenAndServe()
	}()
	srv.ListenAndServe()
}
