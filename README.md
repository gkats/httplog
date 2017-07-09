# gkats/httplog

Package `httplog` provides logging for http requests.

Apart from a ready to use logger that can be used freely, the package also provides a logging middleware (or wrapper) over http Handlers.

## Install

```
$ go get github.com/gkats/httplog
```

## Details

The functionality is purposely kept minimal. The logger outputs a small set of default parameters and provides an extensible way to log extra parameters if needed. The log format is a nice balance between human and machine readability.

The log output is one line per request. The parameters are separated with a blank space while the parameter key and its value are separated by the "=" character. Here's an example log output for a single request.
```
level=I time=2017-07-08T17:08:12UTC ip=193.92.20.19 method=GET path=/logs ua=Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.81 Safari/537.36 status=200 params={}
```

## Default log parameters

This is a complete list of the default parameters that are logged.

Name | Description
-----|-------------
__level__ | The log output level. This will always be set to (I)nfo. There's really no need for any other log levels.
__time__ | The timestamp of the log entry. The date follows the [ISO 8601](https://en.wikipedia.org/wiki/ISO_8601) format (YYYY-MM-DDTHH:mm:ssZ)
__ip__ | The request's IP. Takes into account the `X-Forwarded-For` header if it's set.
__method__ | The request method.
__path__ | The path for the request, leaving out the hostname part.
__ua__ | The request's `User-Agent` header.
__status__ | The response status code.
__params__ | Any parameters that came with the request. The parameters will be logged in JSON format, even for non-JSON requests. The parameters are taken from either the request body or the query parameters (for GET requests).

## Usage

### Standalone

The logger needs a stream that implements the [io.Writer](https://golang.org/pkg/io/#Writer) interface. This is where all logging output will go.
```
type stream struct {}

func (s *stream) Write(p []byte) (n int, err error) {
  // write somewhere
}

l := httplog.New(&stream{})
l.Log()
```

### Middleware

If you're familiar with the `http.Handler` middleware pattern, you just need to provide your handler and a logger as arguments to the `httplog.WithLogging` function.
```
type customHandler {}

func (h *customHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  w.WriteHeader(200)
  // ...your handler logic goes here...
}

func main() {
  // Configure a logger
  l := httplog.New(os.Stdout)

  // And use the middleware
  http.Handle("/logs", httplog.WithLogging(&customHandler{}, l))
  http.ListenAndServe(":8080", nil)
}
```

Performing a request to `GET http://server.url:8080/logs?q=works` will produce the following line in your server's standard output.
```
level=I time=2017-07-08T17:08:12UTC ip=193.92.20.19 method=GET path=/logs ua=Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.81 Safari/537.36 status=200 params={"q": "works"}
```

### Adding extra log parameters

Sometimes you might want to log some extra parameters, like information about the user making requests. You can use the logger's `Add` method and specify the parameter name and the value.
```
type User struct {
  ID int
}
user := &User{ID: 1234}

l := httplog.New(os.Stdout)
l.Add("uid", user.ID)
l.Add("meta", "new-request")
l.Log()

// => level=I [...] uid=1234 meta=new-request
```

## Contributing

Pull requests, bug fixes and issue reports are more than welcome! Please keep in mind that the goal is to keep the functionality minimal.

## License

The package is released under the [MIT License](https://opensource.org/licenses/MIT)