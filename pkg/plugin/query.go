package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
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

// For development purposes
const maxDataValue = 100

type queryModel struct {
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
	maxValue := int64(maxDataValue) // You can adjust this maximum value as needed
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
