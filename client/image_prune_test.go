package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/filters"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestImagesPruneError(t *testing.T) {
	client := &Client{
		client:  newMockClient(errorMock(http.StatusInternalServerError, "Server error")),
		version: "1.25",
	}

	filters := filters.NewArgs()

	_, err := client.ImagesPrune(context.Background(), filters)
	assert.EqualError(t, err, "Error response from daemon: Server error")
}

func TestImagesPrune(t *testing.T) {
	expectedURL := "/v1.25/images/prune"

	danglingFilters := filters.NewArgs()
	danglingFilters.Add("dangling", "true")

	noDanglingFilters := filters.NewArgs()
	noDanglingFilters.Add("dangling", "false")

	labelFilters := filters.NewArgs()
	labelFilters.Add("dangling", "true")
	labelFilters.Add("label", "label1=foo")
	labelFilters.Add("label", "label2!=bar")

	listCases := []struct {
		filters             filters.Args
		expectedQueryParams map[string]string
	}{
		{
			filters: filters.Args{},
			expectedQueryParams: map[string]string{
				"until":   "",
				"filter":  "",
				"filters": "",
			},
		},
		{
			filters: danglingFilters,
			expectedQueryParams: map[string]string{
				"until":   "",
				"filter":  "",
				"filters": `{"dangling":{"true":true}}`,
			},
		},
		{
			filters: noDanglingFilters,
			expectedQueryParams: map[string]string{
				"until":   "",
				"filter":  "",
				"filters": `{"dangling":{"false":true}}`,
			},
		},
		{
			filters: labelFilters,
			expectedQueryParams: map[string]string{
				"until":   "",
				"filter":  "",
				"filters": `{"dangling":{"true":true},"label":{"label1=foo":true,"label2!=bar":true}}`,
			},
		},
	}
	for _, listCase := range listCases {
		client := &Client{
			client: newMockClient(func(req *http.Request) (*http.Response, error) {
				if !strings.HasPrefix(req.URL.Path, expectedURL) {
					return nil, fmt.Errorf("Expected URL '%s', got '%s'", expectedURL, req.URL)
				}
				query := req.URL.Query()
				for key, expected := range listCase.expectedQueryParams {
					actual := query.Get(key)
					assert.Equal(t, expected, actual)
				}
				content, err := json.Marshal(types.ImagesPruneReport{
					ImagesDeleted: []types.ImageDeleteResponseItem{
						{
							Deleted: "image_id1",
						},
						{
							Deleted: "image_id2",
						},
					},
					SpaceReclaimed: 9999,
				})
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader(content)),
				}, nil
			}),
			version: "1.25",
		}

		report, err := client.ImagesPrune(context.Background(), listCase.filters)
		assert.NoError(t, err)
		assert.Len(t, report.ImagesDeleted, 2)
		assert.Equal(t, uint64(9999), report.SpaceReclaimed)
	}
}
