package httplog_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gkats/httplog"
)

type testHandler struct{}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "{}")
}

func TestWithLogging(t *testing.T) {
	w := NewTestWriter()
	l := httplog.New(w)

	h := httplog.WithLogging(&testHandler{}, l)
	ts := httptest.NewServer(h)
	defer ts.Close()

	if _, err := http.Get(ts.URL + "?foo=bar"); err != nil {
		t.Error(err)
	}

	tests := []string{
		"level=I",
		"time=",
		"ip=",
		"method=GET",
		"path=",
		"ua=",
		"status=200",
		"params={\"foo\": \"bar\"}",
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
