// Package tryhttp reschedule failed HTTP requests.
package tryhttp

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/teamwork/utils/stringutil"
)

// Client to perform HTTP requests.
type Client struct {
	// Retry determines if we should retry the request after failure. It will
	// keep retrying until the second argument returns false. The first return
	// argument can be used to delay the HTTP request.
	Retry func(r *http.Request, err error, attempt int) (delay time.Duration, retry bool)

	// Success callback. This will be run after the HTTP request is finished
	// without errors or a non-2xx status code.
	// If the HTTP request runs out of retries then this will never get called.
	// The response Body will be closed after this is run (you don't need to do
	// it in the callback).
	Success func(r *http.Request, resp *http.Response, attempt int)

	// Scheduler to schedule failed requests.
	//
	// This should schedule the request in "delay" seconds by running "c.do(r,
	// attempt+1)". "err" is the error that the previous request returned.
	//
	// If this isn't given ScheduleGoroutine() will be used.
	Scheduler func(c Client, r *http.Request, attempt int, delay time.Duration, err error)

	// Client to perform requests with. If this isn't given a http.Client with a
	// timeout of 10 seconds will be used.
	Client *http.Client
}

var (
	// DefaultClient is a HTTP client with a timeout of 10 seconds.
	DefaultClient = &http.Client{Timeout: 10 * time.Second}
)

// ErrorNotOkay is used when a HTTP request succeeded (e.g. not connection
// error) but did not return a 2xx status code.
type ErrorNotOkay struct {
	Status int    // HTTP status code.
	Body   string // First 300 characters of body.
}

func (err *ErrorNotOkay) Error() string {
	return fmt.Sprintf("status %v: %v", err.Status, err.Body)
}

// New creates a new Request object, ensuring that all blank fields are given
// sane defaults.
func New(c Client) Client {
	if c.Retry == nil {
		panic("Retry is nil!")
	}
	if c.Client == nil {
		c.Client = DefaultClient
	}
	if c.Scheduler == nil {
		c.Scheduler = ScheduleGoroutine
	}
	return c
}

// Do performs the HTTP request; if this fails it will be sent for retry to
// the Qeueue.
func (c Client) Do(r *http.Request) {
	c.do(r, 0)
}

func (c Client) do(r *http.Request, attempt int) {
	resp, err := c.Client.Do(r)

	// Consider non-200 status code to be errors; this won't set err.
	if err == nil && (resp.StatusCode < 200 || resp.StatusCode > 299) {
		b, _ := ioutil.ReadAll(resp.Body)
		err = &ErrorNotOkay{Status: resp.StatusCode, Body: stringutil.Left(string(b), 300)}
	}

	if err == nil {
		if c.Success != nil {
			c.Success(r, resp, attempt)
		}
		// Make sure response body gets closed.
		_ = resp.Body.Close()
		return
	}

	if resp.Body != nil {
		resp.Body.Close() // nolint: errcheck
	}

	delay, retry := c.Retry(r, err, attempt)
	if !retry {
		return
	}

	c.Scheduler(c, r, attempt, delay, err)
}

// Goroutines tracks the number of currently active goroutines started with
// ScheduleGoroutine().
var Goroutines sync.WaitGroup

// ScheduleGoroutine will start a new goroutine that will sleep for delay
// before performing the HTTP request.
//
// Remember that while goroutines are cheap, they are not free! If there is a
// chance that an (extended) network error will cause hundreds of thousands of
// queued requests then this is probably not the method to use.
//
// Also remember that restarting your application will kill goroutines, so those
// scheduled requests will be LOST.
func ScheduleGoroutine(c Client, r *http.Request, attempt int, delay time.Duration, err error) {
	Goroutines.Add(1)
	go func() {
		defer Goroutines.Done()
		time.Sleep(delay)
		c.do(r, attempt+1)
	}()
}
