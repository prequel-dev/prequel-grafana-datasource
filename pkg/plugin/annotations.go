package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type annotationRequest struct {
	Range struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"range"`
	Annotation struct {
		Query string `json:"query"`
	} `json:"annotation"`
}

func setQueryParams(u *url.URL, areq *annotationRequest) *url.URL {
	q := u.Query()
	q.Set("order", `timestamp+`)
	q.Set("filter", fmt.Sprintf("timestamp.>=.%d,timestamp.<=.%d", areq.Range.From.UnixNano(), areq.Range.To.UnixNano()))
	u.RawQuery = q.Encode()
	return u
}

func (d *Datasource) handleAnnotations(_ context.Context, creq *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {

	// Decode the incoming request from Grafana

	log.DefaultLogger.Info("handleAnnotations", "req.Body", string(creq.Body))
	var areq annotationRequest
	if err := json.Unmarshal(creq.Body, &areq); err != nil {
		log.DefaultLogger.Error("handleAnnotations Error", "req.Body", string(creq.Body), "err", err.Error())
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusBadRequest,
			Body:   []byte("failed to unmarshal annotation request: " + err.Error()),
		})
	}

	// Request detections from the Prequel API
	u := setQueryParams(d.url, &areq)

	url := renderURL(u, "detections")
	log.DefaultLogger.With("url", url).Debug("Fetching detections for annotations")

	resp, err := d.httpClient.Get(url)
	if err != nil {
		return sendCallResourceError(sender, "Failed request detections API", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return sendCallResourceError(sender, "Failed to fetch detections from API", fmt.Errorf("status code %d", resp.StatusCode))
	}

	var dresp detectionResponse
	err = json.NewDecoder(resp.Body).Decode(&dresp)
	if err != nil {
		return sendCallResourceError(sender, "Failed to decode detections response", err)
	}

	annotations := []map[string]any{}
	for _, item := range dresp.Items {
		annotations = append(annotations, map[string]interface{}{
			"time":  item.Timestamp / int64(time.Millisecond), // Convert to milliseconds
			"title": item.RuleTitle,
			"text":  fmt.Sprintf(`<a href="https://app-dev.prequel.dev/detections/%s">See full incident report</a>`, item.DetectionID),
			"tags":  []string{item.Category},
		})
	}

	// 3. Marshal the response and send it back to Grafana
	responseJSON, err := json.Marshal(annotations)
	if err != nil {
		return sendCallResourceError(sender, "Failed to marshal annotation response", err)
	}

	return sender.Send(&backend.CallResourceResponse{
		Status: http.StatusOK,
		Body:   responseJSON,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
	})
}

type detectionResponseItem struct {
	Timestamp     int64  `json:"timestamp"`
	RuleTitle     string `json:"rule_title"`
	DetectionID   string `json:"detection_id"`
	Category      string `json:"category"`
	Namespace     string `json:"namespace"`
	ContainerName string `json:"container_name"`
	K8sObject     string `json:"k8s_object"`
}

type detectionResponse struct {
	Items []detectionResponseItem `json:"rows"`
}

func sendCallResourceError(sender backend.CallResourceResponseSender, msg string, errs ...error) error {
	// Log the error details
	if len(errs) > 0 && errs[0] != nil {
		log.DefaultLogger.Error(msg, "error", errs[0].Error())
	}

	return sender.Send(&backend.CallResourceResponse{
		Status: http.StatusBadRequest,
		Body:   []byte(msg),
	})
}
