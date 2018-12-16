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
			checkResponse: checkEmptyWithStatus(http.StatusNotFound),
		},
		{
			name:    "get config path that does not exist",
			request: httptest.NewRequest(http.MethodGet, "http://testrequest/config/does-not-exist", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return new(ignv2_2types.Config), fmt.Errorf("not acceptable")
			},
			checkResponse: checkEmptyWithStatus(http.StatusInternalServerError),
		},
		{
			name:    "get config path that exists",
			request: httptest.NewRequest(http.MethodGet, "http://testrequest/config/master", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return new(ignv2_2types.Config), nil
			},
			checkResponse: checkConfigGet,
		},
		{
			name:    "head config path that exists",
			request: httptest.NewRequest(http.MethodHead, "http://testrequest/config/master", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return new(ignv2_2types.Config), nil
			},
			checkResponse: checkConfigHead,
		},
		{
			name:    "post non-config path that does not exist",
			request: httptest.NewRequest(http.MethodPost, "http://testrequest/post", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return nil, nil
			},
			checkResponse: checkEmptyWithStatus(http.StatusMethodNotAllowed),
		},
		{
			name:    "post config path that exists",
			request: httptest.NewRequest(http.MethodPost, "http://testrequest/config/master", nil),
			serverFunc: func(poolRequest) (*ignv2_2types.Config, error) {
				return new(ignv2_2types.Config), nil
			},
			checkResponse: checkEmptyWithStatus(http.StatusMethodNotAllowed),
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

func checkEmpty(t *testing.T, response *http.Response) {
	contentLength := int(response.ContentLength)
	if contentLength != 0 {
		t.Errorf("expected empty response, but Content-Length was %d", contentLength)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty response, but body length was %d", len(body))
	}
}

func checkNonEmpty(t *testing.T, response *http.Response) {
	contentLength := int(response.ContentLength)
	if contentLength == 0 {
		t.Errorf("expected non-empty response, but Content-Length was 0")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != contentLength {
		t.Errorf("response body length %d does not match Content-Length %d", len(body), contentLength)
	}
}

func checkHeadContent(t *testing.T, response *http.Response) {
	contentLength := int(response.ContentLength)
	if contentLength == 0 {
		t.Errorf("expected non-empty response, but Content-Length was 0")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty HEAD response, but body length was %d", len(body))
	}
}

func checkEmptyWithStatus(status int) checkResponse {
	return func(t *testing.T, response *http.Response) {
		if response.StatusCode != status {
			t.Errorf("expected: %d, received: %d", status, response.StatusCode)
		}

		checkEmpty(t, response)
	}
}

func checkConfig(t *testing.T, response *http.Response) {
	if response.StatusCode != http.StatusOK {
		t.Errorf("expected: %d, received: %d", http.StatusNotFound, response.StatusCode)
	}

	contentType := response.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("response Content-Type is not application/json: %q", contentType)
	}
}

func checkConfigGet(t *testing.T, response *http.Response) {
	checkConfig(t, response)
	checkNonEmpty(t, response)
}

func checkConfigHead(t *testing.T, response *http.Response) {
	checkConfig(t, response)
	checkHeadContent(t, response)
}
