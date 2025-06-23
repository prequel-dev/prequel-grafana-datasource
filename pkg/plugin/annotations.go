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
	"github.com/grafana/grafana-plugin-sdk-go/data"
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

func setQueryParams(u *url.URL, from, to time.Time) *url.URL {
	q := u.Query()
	q.Set("order", `timestamp+`)
	q.Set("filter", fmt.Sprintf("timestamp.>=.%d,timestamp.<=.%d", from.UnixNano(), to.UnixNano()))
	u.RawQuery = q.Encode()
	return u
}

func (d *Datasource) handleAnnotationsQueryx(ctx context.Context, query backend.DataQuery, qm queryModel) backend.DataResponse {
	frame := data.NewFrame("annotations")

	// 1. Add the dedicated `linkURL` field to your frame definition.
	// The field name MUST be exactly "linkURL".
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, []time.Time{}),
		data.NewField("title", nil, []string{}),
		data.NewField("text", nil, []string{}),
		data.NewField("tags", nil, []string{}),
		data.NewField("linkURL", nil, []string{}), // <-- THE CORRECT METHOD
	)

	// --- Example Data ---
	deploymentID := "a1b2c3d4"
	commitSHA := "7f8d9e0"

	// 2. Define the URL. The text field should now be clean, without any Markdown.
	url := fmt.Sprintf("http://localhost:3000/my-org/my-repo/commit/%s", commitSHA)
	description := fmt.Sprintf("Deployment ID %s succeeded. <a href=\"%s\">View the commit for details.</a>", deploymentID, url)

	annotationTime, _ := time.Parse(time.RFC3339, "2025-06-23T14:00:00Z")

	// 3. Add the row, providing the URL as the value for the `linkURL` field.
	frame.AppendRow(
		annotationTime,
		"Production Release v1.2.3", // title
		description,                 // text
		"deploy,production",         // tags
		url,                         // linkURL
	)

	// 4. For an annotation WITHOUT a link, simply provide an empty string for the linkURL.
	alertTime, _ := time.Parse(time.RFC3339, "2025-06-23T15:30:00Z")
	frame.AppendRow(
		alertTime,
		"CPU Alert",
		"High CPU usage detected on server-prod-01.",
		"alert,cpu",
		"", // No link for this annotation
	)

	// 5. Return the DataResponse.
	dataResponse := backend.DataResponse{
		Frames: []*data.Frame{frame},
	}

	return dataResponse
}

func (d *Datasource) handleAnnotationsQuery(_ context.Context, query backend.DataQuery, qm queryModel) backend.DataResponse {
	log.DefaultLogger.Info("handleAnnotationsQuery", "query", query)

	// //log.DefaultLogger.Info("QUERY", "JSON", string(query.JSON), "start", query.TimeRange.From, "end", query.TimeRange.To
	// if err := json.Unmarshal(creq.Body, &areq); err != nil {
	// 	log.DefaultLogger.Error("handleAnnotations Error", "req.Body", string(creq.Body), "err", err.Error())
	// 	return sender.Send(&backend.CallResourceResponse{
	// 		Status: http.StatusBadRequest,
	// 		Body:   []byte("failed to unmarshal annotation request: " + err.Error()),
	// 	})
	// }

	// Request detections from the Prequel API
	u := setQueryParams(d.url, query.TimeRange.From, query.TimeRange.To)

	url := renderURL(u, "detections")
	log.DefaultLogger.With("url", url).Debug("Fetching detections for annotations")

	resp, err := d.httpClient.Get(url)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("Failed request detections API: %v", err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("Failed to fetch detections from API, status code: %d", resp.StatusCode))
	}

	var dresp detectionResponse
	err = json.NewDecoder(resp.Body).Decode(&dresp)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("Failed to decode detections response: %v", err.Error()))
	}

	frame := data.NewFrame("annotations")
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, []int64{}),
		data.NewField("title", nil, []string{}),
		data.NewField("text", nil, []string{}),
		data.NewField("tags", nil, []string{}),
	)

	for _, item := range dresp.Items {
		text := fmt.Sprintf(`<a target="_top" href="https://app-dev.prequel.dev/detections/%s">See full incident report</a>`, item.DetectionID)
		frame.AppendRow(item.Timestamp/int64(time.Millisecond), item.RuleTitle, text, item.Category)
	}

	return backend.DataResponse{
		Status: http.StatusOK,
		Frames: []*data.Frame{frame},
	}
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
	u := setQueryParams(d.url, areq.Range.From, areq.Range.To)

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
