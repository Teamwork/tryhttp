package tryhttp

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/tevino/abool"
)

func Test(t *testing.T) {
	c := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c < 2 {
			w.WriteHeader(http.StatusTeapot)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		c++
		w.Write([]byte("hello")) // nolint: errcheck
	}))

	successCalled := abool.New()
	h := New(Client{
		Retry: func(r *http.Request, n int) (time.Duration, bool) {
			return 200 * time.Millisecond, true
		},
		Success: func(r *http.Request, resp *http.Response, attempt int) {
			successCalled.Set()
			resp.Body.Close() // nolint: errcheck
		},
	})
	u, _ := url.Parse(server.URL)
	h.Do(&http.Request{
		URL: u,
	})

	time.Sleep(250 * time.Millisecond)
	if successCalled.IsSet() {
		t.Fatal("successCalled == true too soon!")
	}

	Goroutines.Wait()
	if !successCalled.IsSet() {
		t.Fatal("successCalled == false")
	}
}
