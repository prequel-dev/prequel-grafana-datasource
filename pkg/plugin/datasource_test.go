package plugin

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestQueryData(t *testing.T) {
	ds := Datasource{}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A", JSON: []byte(`{"query": "SELECT * FROM users"}`)},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	if len(resp.Responses) != 1 {
		t.Fatal("QueryData must return a response")
	}
}

func TestRenderUrl(t *testing.T) {
	u, err := url.Parse("https://app-dev.prequel.dev")
	if err != nil {
		t.Fatal(err)
	}

	q := u.Query()
	q.Set("order", `[{"order":"asc","id":"timestamp"}]`)

	u.RawQuery = q.Encode()
	url := renderURL(u, "detections")

	fmt.Println(url)
}
