package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	Namespace = "aivenator"

	LabelOperation          = "operation"
	LabelNamespace          = "namespace"
	LabelPool               = "pool"
	LabelResourceType       = "resource_type"
	LabelStatus             = "status"
	LabelSyncState          = "synchronization_state"
	LabelProcessingReason   = "processing_reason"
	LabelSecretState        = "state"
	LabelUserNameConvention = "username_convention"
	LabelHandler            = "handler"
)

type Reason string

const (
	HashChanged           Reason = "HashChanged"
	MissingSecret         Reason = "MissingSecret"
	MissingOwnerReference Reason = "MissingOwnerReference"
)

func (r Reason) String() string {
	return string(r)
}

var (
	BigQueryDatasetProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "bqrator_bigquerydataset_processed",
		Namespace: Namespace,
		Help:      "number of bigquerydataset synchronized",
	}, []string{LabelSyncState})
)

func Register(registry prometheus.Registerer) {
	registry.MustRegister(
		BigQueryDatasetProcessed,
	)
}
