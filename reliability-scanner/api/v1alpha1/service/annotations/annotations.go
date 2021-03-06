package annotations

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy-plugins/reliability-scanner/internal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	checkName string     = "annotations"
)

// QuerierSpec defines the Specification for a Querier.
type QuerierSpec struct {
	Key                string `yaml:"key"`
	ValidateURL        bool   `yaml:"validate_url"`
	IncludeAnnotations bool   `yaml:"include_annotations"`
}

// Querier defines the query and set of checks.
type Querier struct {
	client *kubernetes.Clientset
	Spec   *QuerierSpec `yaml:"spec"`
}

// AddtoRunner configures a runner with the Querier for this check.
func (querier *Querier) AddtoRunner(runner *internal.Runner) {
			runner.Queriers = append(runner.Queriers, querier)
			runner.Logger.WithFields(log.Fields{
				"check_name": checkName,
				"phase": "add",
			}).Info("complete")
}

// NewQuerier returns a new configured Querier.
func NewQuerier(spec *QuerierSpec) (Querier, error) {
	out := Querier{
		Spec: spec,
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return out, err
	}
	out.client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return out, err
	}
	return out, nil
}

// Start runs the Querier.
func (q Querier) Start(cfg *internal.QuerierConfig) {
	cfg.Logger.WithFields(log.Fields{
		"check_name": checkName,
		"phase": "add",
	}).Info(internal.CheckStartMsg)

	checkItem := internal.ReportItem{
		Name:   checkName,
		Status: "passed",
	}

	services, err := q.client.CoreV1().Services("").List(cfg.Context, metav1.ListOptions{})
	if err != nil {
		checkItem.Status = "failed"
	}

	for _, service := range services.Items {

		details := make(map[string]interface{})

		if service.ObjectMeta.Labels["sonobuoy-component"] == "aggregator" {
			continue
		}

		if q.Spec.IncludeAnnotations {
			annotations := make(map[string]interface{})
			for k, v := range service.Labels {
				annotations[k] = v
			}
			details["annotations"] = annotations
		}

		item := internal.Item{
			Name:    service.ObjectMeta.Name,
			Status:  "passed",
			Details: details,
		}

		if strings.TrimSpace(q.Spec.Key) != "" {
			if _, exists := service.ObjectMeta.Annotations[q.Spec.Key]; !exists {
				checkItem.Status = "failed"
				item.Status = "failed"
				item.Details["error"] = fmt.Sprintf("key: %s does not exist", q.Spec.Key)
			}
		}
		checkItem.Items = append(checkItem.Items, item)
	}

	cfg.Logger.WithFields(log.Fields{
		"component": "check",
		"check_name": checkName,
		"phase": "complete",
	}).Info(internal.CheckCompleteMsg)

	cfg.Results <- checkItem
	
	cfg.Logger.WithFields(log.Fields{
		"component": "check",
		"check_name": checkName,
		"phase": "write",
	}).Info(internal.CheckWriteMsg)
}
