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

	successCalled := new(int32)
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
	h.Do(&http.Request{URL: u})

	time.Sleep(250 * time.Millisecond)
	if atomic.LoadInt32(successCalled) == 1 {
		t.Fatal("successCalled == true too soon!")
	}

	Goroutines.Wait()
	if atomic.LoadInt32(successCalled) != 1 {
		t.Fatal("successCalled == false")
	}
}

func TestSimpleRetry(t *testing.T) {
	// Retry 3 times.
	r := func(r *http.Request, err error, attempt int) (delay time.Duration, retry bool) {
		return 50 * time.Millisecond, attempt <= 3
	}
	_, ok := r(nil, nil, 1)
	if !ok {
		t.Error("wanted !ok for 1")
	}
	_, ok = r(nil, nil, 3)
	if !ok {
		t.Error("wanted !ok for 3")
	}
	_, ok = r(nil, nil, 4)
	if ok {
		t.Error("wanted ok for 4")
	}

	called := new(int32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.StoreInt32(called, atomic.LoadInt32(called)+1)
		w.WriteHeader(http.StatusTeapot)
	}))

	h := New(Client{Retry: r})
	u, _ := url.Parse(server.URL)
	h.Do(&http.Request{URL: u})

	Goroutines.Wait()
	c := atomic.LoadInt32(called)
	if c != 3 {
		t.Errorf("called %v times, not 3", c)
	}
}
