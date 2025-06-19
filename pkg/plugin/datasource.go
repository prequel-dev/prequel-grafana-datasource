package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/prequel/prequel/pkg/models"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces - only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ backend.CallResourceHandler   = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

const (
	defaultHttpTimeout = 30 * time.Second
	urlBase            = "v1/api"
)

// NewDatasource creates a new datasource instance.
func NewDatasource(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &Datasource{}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct{}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	Constant int64 `json:"constant,omitempty"`
}

func (d *Datasource) query(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	log.DefaultLogger.Info("QUERY", "JSON", string(query.JSON), "start", query.TimeRange.From, "end", query.TimeRange.To)

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	// create data frame response.
	// For an overview on data frames and how grafana handles them:
	// https://grafana.com/developers/plugin-tools/introduction/data-frames
	frame := data.NewFrame("response")

	// Generate evenly distributed time points (example with 10 points)
	timePoints := populateTimePoints(query.TimeRange, 10)

	// Generate random values between 0 and maxValue
	maxValue := int64(qm.Constant) // You can adjust this maximum value as needed
	values := make([]int64, len(timePoints))
	for i := range values {
		values[i] = rand.Int63n(maxValue + 1) // rand.Int63n(n) returns [0, n)
	}

	// add fields.
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, timePoints),
		data.NewField("values", nil, values),
	)

	// add the frames to the response.
	response.Frames = append(response.Frames, frame)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(_ context.Context, creq *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	config, err := models.LoadPluginSettings(*creq.PluginContext.DataSourceInstanceSettings)

	if err != nil {
		return renderHealthCheckError("Failed to load plugin settings", err), nil
	}

	if config.URL == "" {
		return renderHealthCheckError("API URL is missing"), nil
	}

	u, err := url.Parse(config.URL)
	if err != nil {
		return renderHealthCheckError("Failed to parse API URL", err), nil
	}

	if config.Secrets.Token == "" {
		return renderHealthCheckError("Token is missing"), nil
	}

	url := renderURL(u, "health")
	log.DefaultLogger.With("url", url).Debug("Checking health of prequel datasource")

	req, err := createHttpRequest("GET", url, config.Secrets.Token)
	if err != nil {
		return renderHealthCheckError("Failed to create health check request", err), nil
	}

	resp, err := executeHttpRequest(req)
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

func renderURL(u *url.URL, upath string) string {
	u = u.JoinPath(urlBase, upath)
	return u.String()
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

func createHttpRequest(method, url string, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func executeHttpRequest(req *http.Request) (*http.Response, error) {
	client := &http.Client{Timeout: defaultHttpTimeout}
	return client.Do(req)
}

func (d *Datasource) CallResource(ctx context.Context, creq *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	if creq.Path == "annotations" {
		return d.handleAnnotations(ctx, creq, sender)
	}

	return sender.Send(&backend.CallResourceResponse{
		Status: http.StatusNotFound,
	})
}

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
	config, err := models.LoadPluginSettings(*creq.PluginContext.DataSourceInstanceSettings)

	if err != nil {
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusNotFound,
			Body:   []byte("failed to load plugin settings: " + err.Error()),
		})
	}

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
	u, err := url.Parse(config.URL)
	if err != nil {
		return sendCallResourceError(sender, "Failed to parse API URL", err)
	}

	u = setQueryParams(u, &areq)

	url := renderURL(u, "detections")
	log.DefaultLogger.With("url", url).Debug("Fetching detections for annotations")
	req, err := createHttpRequest("GET", url, config.Secrets.Token)
	if err != nil {
		return sendCallResourceError(sender, "Failed to create detections request", err)
	}

	resp, err := executeHttpRequest(req)
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

	annotations := []map[string]interface{}{}
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

// populateTimePoints generates a specified number of evenly distributed time points
// between the given TimeRange.From and TimeRange.To
// Fake data for POC only
func populateTimePoints(timeRange backend.TimeRange, numPoints int) []time.Time {
	if numPoints <= 0 {
		return []time.Time{}
	}

	if numPoints == 1 {
		// If only one point is requested, return the midpoint
		duration := timeRange.To.Sub(timeRange.From)
		midPoint := timeRange.From.Add(duration / 2)
		return []time.Time{midPoint}
	}

	points := make([]time.Time, numPoints)
	totalDuration := timeRange.To.Sub(timeRange.From)

	// Calculate the interval between points
	// We use (numPoints - 1) to ensure the last point is exactly at TimeRange.To
	interval := totalDuration / time.Duration(numPoints-1)

	for i := 0; i < numPoints; i++ {
		if i == numPoints-1 {
			// Ensure the last point is exactly TimeRange.To
			points[i] = timeRange.To
		} else {
			points[i] = timeRange.From.Add(time.Duration(i) * interval)
		}
	}

	return points
}
