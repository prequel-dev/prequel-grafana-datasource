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

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// b, _ := json.Marshal(req)
	// log.DefaultLogger.Info("QueryData", "req", string(b))
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
	QueryText string `json:"queryText"`
	QueryType string `json:"queryType"`
}

func (d *Datasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	if query.QueryType == "annotations" {
		return d.queryAnnotations(ctx, pCtx, query)
	}

	return d.queryData(ctx, pCtx, query)
}

func (d *Datasource) queryData(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var response backend.DataResponse

	// create data frame response.
	// For an overview on data frames and how grafana handles them:
	// https://grafana.com/developers/plugin-tools/introduction/data-frames
	frame := data.NewFrame("response")

	// add fields.
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, []time.Time{query.TimeRange.From, query.TimeRange.To}),
		data.NewField("values", nil, []int64{10, 20}),
	)

	// add the frames to the response.
	response.Frames = append(response.Frames, frame)

	return response
}

// The queryText is expected to be a string, currently of the format of the filter expression in the URL
// e.g. "cluster_name.==.aleks-jun13,k8s_object.==.oom-demo2"
func setQueryParams(u *url.URL, from, to time.Time, queryText string) *url.URL {
	newUrl := *u
	q := newUrl.Query()
	q.Set("order", `timestamp+`)
	filter := fmt.Sprintf("timestamp.>=.%d,timestamp.<=.%d", from.UnixNano(), to.UnixNano())
	if queryText != "" {
		filter += fmt.Sprintf(",%s", queryText)
	}
	q.Set("filter", filter)
	newUrl.RawQuery = q.Encode()
	return &newUrl
}

func (d *Datasource) queryAnnotations(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	log.DefaultLogger.Info("queryAnnotations", "queryText", qm.QueryText)

	u := setQueryParams(d.url, query.TimeRange.From, query.TimeRange.To, qm.QueryText)

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
		text := fmt.Sprintf(`<a href="%s/detections/%s">See full incident report</a>`, d.url.String(), item.DetectionID)
		frame.AppendRow(item.Timestamp/int64(time.Millisecond), item.RuleTitle, text, item.Category)
	}

	return backend.DataResponse{
		Status: http.StatusOK,
		Frames: []*data.Frame{frame},
	}
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
