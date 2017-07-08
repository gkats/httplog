package httplog

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Logger interface {
	Log()
	SetStatus(int)
	SetRequestInfo(*http.Request)
}

type httpLogger struct {
	w      io.Writer
	ip     string
	method string
	path   string
	ua     string
	params string
	status int
	reqRaw []byte
}

func New(w io.Writer) Logger {
	return &httpLogger{w: w}
}

func (l *httpLogger) Log() {
	l.w.Write(append(l.buildLogEntry(), '\n'))
}

func (l *httpLogger) buildLogEntry() []byte {
	buf := make([]byte, 0)
	buf = append(buf, "level=I"...)
	buf = append(buf, " time="+time.Now().UTC().Format("2006-01-02T15:04:05MST")...)
	buf = append(buf, " ip="+l.ip...)
	buf = append(buf, " method="+l.method...)
	buf = append(buf, " path="+l.path...)
	buf = append(buf, " ua="+l.ua...)
	buf = append(buf, " status="+strconv.Itoa(l.status)...)
	buf = append(buf, " params="+l.params...)
	return buf
}

func (l *httpLogger) SetStatus(s int) {
	l.status = s
}

func (l *httpLogger) SetRequestInfo(r *http.Request) {
	l.ip = getIP(r)

	// Get a request dump
	l.reqRaw = reqDump(r)

	var line string
	pathRegexp, _ := regexp.Compile("(.+)\\s(.+)\\sHTTP")
	userAgentRegexp, _ := regexp.Compile("User-Agent:\\s(.+)")
	getParamsRegexp, _ := regexp.Compile("(.+)\\?(.+)")

	// The raw request comes in lines, separated by \r\n
	s := bufio.NewScanner(strings.NewReader(string(l.reqRaw)))
	for s.Scan() {
		line = s.Text()
		l.setPath(line, pathRegexp, getParamsRegexp)
		l.setUa(line, userAgentRegexp)
	}
	// Last line contains the request parameters
	if len(l.params) == 0 {
		l.params = line
	}
}

func (l *httpLogger) setPath(path string, pathRegexp *regexp.Regexp, getParamsRegexp *regexp.Regexp) {
	// Check for the request path portion
	// example POST /path HTTP/1.1
	matches := pathRegexp.FindStringSubmatch(path)
	if len(matches) > 0 {
		l.method = matches[1]
		l.path = matches[2]
		// Check for query string params (GET request)
		// example GET /path?param1=value&param2=value
		matches = getParamsRegexp.FindStringSubmatch(matches[2])
		if len(matches) > 0 {
			l.path = matches[1]
			l.params = toJSON(matches[2])
		}
	}
}

func (l *httpLogger) setUa(h string, r *regexp.Regexp) {
	// Check for user agent header
	// example User-Agent: <ua>
	if matches := r.FindStringSubmatch(h); len(matches) > 0 {
		l.ua = matches[1]
	}
}

func getIP(r *http.Request) (ip string) {
	if forwarded := r.Header.Get("X-Forwarded-For"); len(forwarded) > 0 {
		ip = forwarded
		return
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return
}

func reqDump(r *http.Request) (dump []byte) {
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		dump = []byte("")
	}
	return
}

// Poor man's JSON encoding
func toJSON(s string) string {
	r := strings.NewReplacer("=", "\": \"", "&", "\", \"")
	return "{\"" + r.Replace(s) + "\"}"
}
