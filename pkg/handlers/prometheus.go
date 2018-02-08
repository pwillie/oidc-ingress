package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	reqInFlightGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_in_flight_requests",
		Help: "A gauge of requests currently being served by the wrapped handler.",
	})

	reqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "A counter for requests to the wrapped handler.",
		},
		[]string{"code", "method"},
	)

	// duration is partitioned by the HTTP method and handler. It uses custom
	// buckets based on the expected request duration.
	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_milliseconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"code", "method"},
	)
)

func init() {
	// Register all of the metrics in the standard registry.
	prometheus.MustRegister(reqInFlightGauge, reqCounter, reqDuration)
}

func PrometheusHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			reqInFlightGauge.Inc()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				// entry.Write(ww.Status(), ww.BytesWritten(), time.Since(t1))
				status := strconv.Itoa(ww.Status())
				reqCounter.WithLabelValues(status, r.Method).Inc()
				stop := time.Now()
				l := stop.Sub(t1)
				reqDuration.WithLabelValues(status, r.Method).Observe(float64(l) / 1000000)
				reqInFlightGauge.Dec()
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}
