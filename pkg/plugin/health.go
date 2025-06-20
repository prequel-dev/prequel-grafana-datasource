package plugin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(_ context.Context, creq *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	url := renderURL(d.url, "health")

	resp, err := d.httpClient.Get(url)
	if err != nil {
		return renderHealthCheckError("Failed to connect to health check API", err), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return renderHealthCheckError(fmt.Sprintf("API returned status code %d", resp.StatusCode)), nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}

func renderHealthCheckError(msg string, errs ...error) *backend.CheckHealthResult {
	// Log the error details
	if len(errs) > 0 && errs[0] != nil {
		log.DefaultLogger.Error(msg, "error", errs[0].Error())
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusError,
		Message: msg,
	}
}
