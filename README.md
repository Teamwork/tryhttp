[![Build Status](https://travis-ci.org/Teamwork/tryhttp.svg?branch=master)](https://travis-ci.org/Teamwork/tryhttp)
[![codecov](https://codecov.io/gh/Teamwork/tryhttp/branch/master/graph/badge.svg?token=n0k8YjbQOL)](https://codecov.io/gh/Teamwork/tryhttp)
[![GoDoc](https://godoc.org/github.com/Teamwork/tryhttp?status.svg)](https://godoc.org/github.com/Teamwork/tryhttp)

Go library to reschedule failed HTTP requests.

Communicating to services over HTTP is unreliably; a service may be temporarily
down, there may be a network error, DNS error, or something else may go wrong.

This package implements an interface to retry failed requests, either with
goroutines or by queuing them in a more persistent queue system (such as
RabbitMQ or Redis).

A simple example:

```go
r := tryhttp.New(tryhttp.Client{
    // Determine if we should retry.
    Retry: func(r *http.Request, err error, attempt int) (delay time.Duration, retry bool) {
        retry = attempt < 2
        if !retry {
            log.Print("Aborting request to %s", r.URL)
        }
        return time.Second*10, retry
    }

    // Schedule failed messages with this.
    Scheduler: tryhttp.ScheduleGoroutine,
})

req, err := http.NewRequest("GET", "http://example.com", nil)
if err != nil {
    log.Fatal(err)
}
r.Do(req)
```

Why not just use goroutines?
----------------------------

Goroutines stop when the application exits, which is okay in some cases, but
sometimes it's not.

Could this be extended beyond HTTP requests?
--------------------------------------------

Most certainly; but limiting it to HTTP requests keeps the API simple and
`interface{}`-free.
