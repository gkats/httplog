// Package httplog provides logging for http requests.
//
// Apart from a ready to use logger that can be used freely, the package also
// provides a logging middleware (or wrapper) over http Handlers.
//
// The logger outputs a small set of default parameters and provides an
// extensible way to log extra parameters if needed. The log format is a nice
// balance between human and machine readability.
//
// The log output is one line per request. The parameters are separated with a
// blank space while the parameter key and its value are separated by the "="
// character. Here's an example log output for a single request.
//
//   level=I time=2017-07-08T17:08:12UTC ip=193.92.20.19 method=GET path=/logs ua=Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.81 Safari/537.36 status=200 params={}
//
// Standalone usage
//
// The logger needs a stream that implements the io.Writer interface. This is where
// all logging output will go.
//
//   type stream struct {}
//
//   func (s *stream) Write(p []byte) (n int, err error) {
//     // write somewhere
//   }
//
//   l := httplog.New(&stream{})
//   l.Log()
//
// Middleware usage
//
// You just need to provide your handler and a logger as arguments to the
// httplog.WithLogging function.
//
//   type customHandler {}
//
//   func (h *customHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//     w.WriteHeader(200)
//     // ...your handler logic goes here...
//   }
//
//   func main() {
//     // Configure a logger
//     l := httplog.New(os.Stdout)
//
//     // And use the middleware
//     http.Handle("/logs", httplog.WithLogging(&customHandler{}, l))
//     http.ListenAndServe(":8080", nil)
//   }
//
// Adding extra log parameters
//
//   type User struct {
//     ID int
//   }
//   user := &User{ID: 1234}
//
//   l := httplog.New(os.Stdout)
//   l.Add("uid", user.ID)
//   l.Add("meta", "new-request")
//   l.Log()
//   // => level=I [...] uid=1234 meta=new-request
//
package httplog

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Logger is the interface that wraps the basic Log method.
// Log builds a log entry with all the parameters the Logger has been
// configured with and writes the log entry to the underlying
// io.Writer's stream.
//
// The Logger can be configured with information from the request, using
// the SetRequestInfo method. After the request is made, the SetStatus
// method will set the response status.
//
// Logger also provides an Add method, to add extra parameters for
// logging.
type Logger interface {
	Log()
	SetStatus(int)
	SetRequestInfo(*http.Request)
	Add(string, interface{})
}

// Concrete implementation of the Logger interface
type httpLogger struct {
	w      io.Writer
	ip     string
	method string
	path   string
	ua     string
	params string
	status int
	reqRaw []byte
	extras map[string]interface{}
}

// New returns a Logger configured with the supplied io.Writer.
func New(w io.Writer) Logger {
	return &httpLogger{
		w:      w,
		extras: make(map[string]interface{}, 0),
	}
}

// Log produces the logging entry for a single request.
// It appends a logging line to the io.Writer's stream, using the io.Writer's
// Write function. The entry is terminated by the new line character.
func (l *httpLogger) Log() {
	l.w.Write(append(l.buildLogEntry(), '\n'))
}

// Add adds an extra logging parameter that will be included in the log output.
// It needs the name of the parameter and its value. The output will be
//   <name>=<value>
func (l *httpLogger) Add(key string, value interface{}) {
	l.extras[key] = value
}

// SetStatus sets the Logger's status field to the supplied value.
func (l *httpLogger) SetStatus(s int) {
	l.status = s
}

// SetRequestInfo sets all Logger fields that can be extracted from the
// supplied http.Request argument. These are:
// - the request IP
// - the request method
// - the user agent header value
// - the path
// - the request parameters, either from the request body or from the query URL
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
	for k, v := range l.extras {
		buf = append(buf, " "+k+"="+fmt.Sprintf("%v", v)...)
	}
	return buf
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
