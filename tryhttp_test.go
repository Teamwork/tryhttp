package tryhttp

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
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

	suc := int32(0)
	successCalled := &suc
	h := New(Client{
		Retry: func(r *http.Request, err error, n int) (time.Duration, bool) {
			return 200 * time.Millisecond, true
		},
		Success: func(r *http.Request, resp *http.Response, attempt int) {
			atomic.StoreInt32(successCalled, 1)
			resp.Body.Close() // nolint: errcheck
		},
	})
	u, _ := url.Parse(server.URL)
	h.Do(&http.Request{
		URL: u,
	})

	time.Sleep(250 * time.Millisecond)
	if atomic.LoadInt32((*int32)(successCalled)) == 1 {
		t.Fatal("successCalled == true too soon!")
	}

	Goroutines.Wait()
	if atomic.LoadInt32((*int32)(successCalled)) != 1 {
		t.Fatal("successCalled == false")
	}
}
