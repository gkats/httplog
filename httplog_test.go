package httplog_test

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gkats/httplog"
)

type testWriter struct {
	Stream string
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	buf := bytes.NewBuffer([]byte(w.Stream))
	buf.Write(p)
	w.Stream = buf.String()
	return
}

func (w *testWriter) Flush() {
	w.Stream = ""
}

func NewTestWriter() *testWriter {
	return &testWriter{Stream: ""}
}

func TestLog(t *testing.T) {
	w := NewTestWriter()
	l := httplog.New(w)
	l.Log()

	tests := []string{
		"level=I",
		"time=",
		"ip=",
		"method=",
		"path=",
		"ua=",
		"status=",
		"params=",
		"\n",
	}
	count := 0
	for i, want := range tests {
		count = strings.Count(w.Stream, want)
		if count != 1 {
			t.Errorf("(%v) Expected %v to contain '%v' exactly once, got %v", i, w.Stream, want, count)
		}
		count = 0
	}

	// Test that it appends to the stream
	l.Log()
	for i, want := range tests {
		count = strings.Count(w.Stream, want)
		if count != 2 {
			t.Errorf("(%v) Expected %v to contain %v exactly twice, got %v", i, w.Stream, want, count)
		}
		count = 0
	}
}

func TestSetStatus(t *testing.T) {
	w := NewTestWriter()
	l := httplog.New(w)
	l.SetStatus(200)
	l.Log()

	want := "status=200"
	if !strings.Contains(w.Stream, want) {
		t.Errorf("Expected %v to include %v", w.Stream, want)
	}
}

func TestSetRequestInfo(t *testing.T) {
	w := NewTestWriter()
	l := httplog.New(w)
	r := httptest.NewRequest("POST", "https://example.com/resources", strings.NewReader("{\"foo\": \"bar\"}"))
	r.Header.Set("User-Agent", "request-ua")
	l.SetRequestInfo(r)

	l.Log()
	tests := []string{
		"ip=" + strings.Split(r.RemoteAddr, ":")[0],
		"method=POST",
		"path=https://example.com/resources",
		"ua=request-ua",
		"params={\"foo\": \"bar\"}",
	}
	for i, want := range tests {
		if !strings.Contains(w.Stream, want) {
			t.Errorf("(%v) Expected %v to contain '%v'", i, w.Stream, want)
		}
	}

	w.Flush()
	forwardedIP := "127.0.0.1"
	r = httptest.NewRequest("GET", "https://example.com/resources?foo=bar", nil)
	r.Header.Set("User-Agent", "request-ua")
	r.Header.Set("X-Forwarded-For", forwardedIP)
	l.SetRequestInfo(r)
	l.Log()

	tests = []string{
		"ip=" + forwardedIP,
		"method=GET",
		"path=https://example.com/resources",
		"ua=request-ua",
		"params={\"foo\": \"bar\"}",
	}
	for i, want := range tests {
		if !strings.Contains(w.Stream, want) {
			t.Errorf("(%v) Expected %v to contain '%v'", i, w.Stream, want)
		}
	}
}

func TestLogExtras(t *testing.T) {
	w := NewTestWriter()
	l := httplog.New(w)
	l.Add("uid", 1234)
	l.Add("secret", "shhh!")
	l.Log()

	tests := []string{
		"level=I",
		"time=",
		"ip=",
		"method=",
		"path=",
		"ua=",
		"status=",
		"params=",
		"uid=1234",
		"secret=shhh!",
		"\n",
	}
	count := 0
	for i, want := range tests {
		count = strings.Count(w.Stream, want)
		if count != 1 {
			t.Errorf("(%v) Expected %v to contain '%v' exactly once, got %v", i, w.Stream, want, count)
		}
		count = 0
	}
}
