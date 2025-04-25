package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	BigQueryDatasetProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bqrator_bigquerydataset_processed",
		Help: "number of bigquerydataset synchronized",
	})
)

func Register(registry prometheus.Registerer) {
	registry.MustRegister(
		BigQueryDatasetProcessed,
	)
}
