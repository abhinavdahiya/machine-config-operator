package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	ignv2_2types "github.com/coreos/ignition/config/v2_2/types"
)

type mockServer struct {
	GetConfigFn func(poolRequest) (*ignv2_2types.Config, error)
}

func (ms *mockServer) GetConfig(pr poolRequest) (*ignv2_2types.Config, error) {
	return ms.GetConfigFn(pr)
}

type checkResponse func(t *testing.T, response *http.Response)

type scenario struct {
	name          string
	request       *http.Request
	serverFunc    func(poolRequest) (*ignv2_2types.Config, error)
	checkResponse checkResponse
}

func TestAPIHandler(t *testing.T) {
	scenarios := []scenario{
		{
			name:    "get non-config path that does not exist",
			request: httptest.NewRequest(http.MethodGet, "http://testrequest/does-not-exist", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return nil, nil
			},
			checkResponse: func(t *testing.T, response *http.Response) {
				checkStatus(t, response, http.StatusNotFound)
				checkContentLength(t, response, 0)
				checkBodyLength(t, response, 0)
			},
		},
		{
			name:    "get config path that does not exist",
			request: httptest.NewRequest(http.MethodGet, "http://testrequest/config/does-not-exist", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return new(ignv2_2types.Config), fmt.Errorf("not acceptable")
			},
			checkResponse: func(t *testing.T, response *http.Response) {
				checkStatus(t, response, http.StatusInternalServerError)
				checkContentLength(t, response, 0)
				checkBodyLength(t, response, 0)
			},
		},
		{
			name:    "get config path that exists",
			request: httptest.NewRequest(http.MethodGet, "http://testrequest/config/master", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return new(ignv2_2types.Config), nil
			},
			checkResponse: func(t *testing.T, response *http.Response) {
				checkStatus(t, response, http.StatusOK)
				checkContentLength(t, response, 114)
				contentType := response.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("response Content-Type is not application/json: %q", contentType)
				}
				checkBodyLength(t, response, 114)
			},
		},
		{
			name:    "head config path that exists",
			request: httptest.NewRequest(http.MethodHead, "http://testrequest/config/master", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return new(ignv2_2types.Config), nil
			},
			checkResponse: func(t *testing.T, response *http.Response) {
				checkStatus(t, response, http.StatusOK)
				checkContentLength(t, response, 114)
				checkBodyLength(t, response, 0)
			},
		},
		{
			name:    "post non-config path that does not exist",
			request: httptest.NewRequest(http.MethodPost, "http://testrequest/post", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return nil, nil
			},
			checkResponse: func(t *testing.T, response *http.Response) {
				checkStatus(t, response, http.StatusMethodNotAllowed)
				checkContentLength(t, response, 0)
				checkBodyLength(t, response, 0)
			},
		},
		{
			name:    "post config path that exists",
			request: httptest.NewRequest(http.MethodPost, "http://testrequest/config/master", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return new(ignv2_2types.Config), nil
			},
			checkResponse: func(t *testing.T, response *http.Response) {
				checkStatus(t, response, http.StatusMethodNotAllowed)
				checkContentLength(t, response, 0)
				checkBodyLength(t, response, 0)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ms := &mockServer{
				GetConfigFn: scenario.serverFunc,
			}
			handler := NewServerAPIHandler(ms)
			handler.ServeHTTP(w, scenario.request)

			resp := w.Result()
			defer resp.Body.Close()
			scenario.checkResponse(t, resp)
		})
	}
}

func checkStatus(t *testing.T, response *http.Response, status int) {
	if response.StatusCode != status {
		t.Errorf("expected: %d, received: %d", status, response.StatusCode)
	}
}

func checkContentLength(t *testing.T, response *http.Response, l int) {
	if int(response.ContentLength) != l {
		t.Errorf("expected response's Content-Length %d, but Content-Length was %d", l, int(response.ContentLength))
	}
}

func checkBodyLength(t *testing.T, response *http.Response, l int) {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != l {
		t.Errorf("expected response's body length to be %d, but body length was %d", l, len(body))
	}
}
