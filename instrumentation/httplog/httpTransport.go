package httplog

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"libs/logger"
)

// NewLogTransport - Creates RoundTripper that logs every HTTP request/response
// Deprecated - Use Options Instead
func NewLogTransport(next http.RoundTripper, logReqBody, logRespBody, passContextFurther bool) http.RoundTripper {
	return &httpLogTransport{
		nextTransport: next,
		defaultSetting: settings{
			Request:      true,
			RequestBody:  logReqBody,
			Response:     true,
			ResponseBody: logRespBody,
			PassContext:  passContextFurther,
		},
	}
}

// Option for LogTransportLogging
type Option interface {
	Apply(transport *httpLogTransport)
}

func New(next http.RoundTripper, opts ...Option) *httpLogTransport {
	t := httpLogTransport{nextTransport: next,
		defaultSetting: defaultLogSetting,
		blackList:      make(map[string]settings),
	}

	for i := range opts {
		opts[i].Apply(&t)
	}

	return &t
}

type endpoint struct {
	Method string
	URL    string
}

func reqKey(method, path string) string {
	return strings.ToLower(method + strings.Trim(path, "/"))
}

func (e *endpoint) toKey() string {
	return reqKey(e.Method, e.URL)
}

func buildBlackList(method string, path []string) []endpoint {
	bl := make([]endpoint, len(path))
	for i := range path {
		bl[i] = endpoint{
			Method: method,
			URL:    path[i],
		}
	}

	return bl
}

func SkipLog(method string, paths ...string) Option {
	return &skipOption{blackList: buildBlackList(method, paths)}
}

type skipOption struct {
	blackList []endpoint
}

func (opt *skipOption) Apply(transport *httpLogTransport) {
	for i := range opt.blackList {
		key := opt.blackList[i].toKey()
		transport.blackList[key] = settings{
			Request:      false,
			RequestBody:  false,
			Response:     false,
			ResponseBody: false,
			PassContext:  false,
		}
	}
}

func SkipReqBody(method string, paths ...string) Option {
	return &skipReqBody{blackList: buildBlackList(method, paths)}
}

type skipReqBody struct {
	blackList []endpoint
}

func (opt *skipReqBody) Apply(transport *httpLogTransport) {
	for i := range opt.blackList {
		key := opt.blackList[i].toKey()
		prev, ok := transport.blackList[key]
		if !ok {
			prev = settings{
				Request:      true,
				RequestBody:  true,
				Response:     true,
				ResponseBody: true,
				PassContext:  true,
			}
		}
		prev.RequestBody = false

		transport.blackList[key] = prev
	}
}

func SkipRespBody(method string, paths ...string) Option {
	return &skipResponseBody{blackList: buildBlackList(method, paths)}
}

type skipResponseBody struct {
	blackList []endpoint
}

func (opt *skipResponseBody) Apply(transport *httpLogTransport) {
	for i := range opt.blackList {
		key := opt.blackList[i].toKey()
		prev, ok := transport.blackList[key]
		if !ok {
			prev = settings{
				Request:      true,
				RequestBody:  true,
				Response:     true,
				ResponseBody: true,
				PassContext:  true,
			}
		}
		prev.RequestBody = false

		transport.blackList[key] = prev
	}
}

func SkipRequest(method string, paths ...string) Option {
	return &skipRequest{blackList: buildBlackList(method, paths)}
}

type skipRequest struct {
	blackList []endpoint
}

func (opt *skipRequest) Apply(transport *httpLogTransport) {
	for i := range opt.blackList {
		key := opt.blackList[i].toKey()
		prev, ok := transport.blackList[key]
		if !ok {
			prev = settings{
				Request:      true,
				RequestBody:  true,
				Response:     true,
				ResponseBody: true,
				PassContext:  true,
			}
		}
		prev.RequestBody = false
		prev.Request = false

		transport.blackList[key] = prev
	}
}

func SkipResponse(method string, paths ...string) Option {
	return &skipReqBody{blackList: buildBlackList(method, paths)}
}

type skipResponse struct {
	blackList []endpoint
}

func (opt *skipResponse) Apply(transport *httpLogTransport) {
	for i := range opt.blackList {
		key := opt.blackList[i].toKey()
		prev, ok := transport.blackList[key]
		if !ok {
			prev = settings{
				Request:      true,
				RequestBody:  true,
				Response:     true,
				ResponseBody: true,
				PassContext:  true,
			}
		}
		prev.ResponseBody = false
		prev.Response = false

		transport.blackList[key] = prev
	}
}

func NoContext(method string, paths ...string) Option {
	return &skipReqBody{blackList: buildBlackList(method, paths)}
}

type noContextPass struct {
	blackList []endpoint
}

func (opt *noContextPass) Apply(transport *httpLogTransport) {
	for i := range opt.blackList {
		key := opt.blackList[i].toKey()
		prev, ok := transport.blackList[key]
		if !ok {
			prev = settings{
				Request:      true,
				RequestBody:  true,
				Response:     true,
				ResponseBody: true,
				PassContext:  true,
			}
		}
		prev.PassContext = false

		transport.blackList[key] = prev
	}
}

type settings struct {
	Request      bool
	RequestBody  bool
	Response     bool
	ResponseBody bool
	PassContext  bool
}

var defaultLogSetting = settings{
	Request:      true,
	RequestBody:  true,
	Response:     true,
	ResponseBody: true,
	PassContext:  true,
}

type httpLogTransport struct {
	nextTransport  http.RoundTripper
	blackList      map[string]settings
	defaultSetting settings
}

func (t *httpLogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	l := logger.FromCtx(ctx, "httplog")

	setting, ok := t.blackList[reqKey(req.Method, req.URL.Path)]
	if !ok {
		setting = t.defaultSetting
	}

	if setting.PassContext {
		req.Header.Set(logger.HTTPHeaderProcessID, logger.GetProcessID(ctx))
		req.Header.Set(logger.HTTPHeaderRequestID, uuid.NewString()) // generates request id header
	}

	fields := make([]zap.Field, 0, 3)
	errs := make([]error, 0, 2)

	if setting.Request {
		body, err := httputil.DumpRequestOut(req, setting.RequestBody)
		if err == nil {
			fields = append(fields, logger.RequestDump(body))
		} else {
			errs = append(errs, err)
		}
	} else {
		fields = append(fields, zap.String(logger.RequestDumpKey, "hidden"))
	}

	resp, err := t.nextTransport.RoundTrip(req)
	if err != nil {
		errs = append(errs, err)
	}

	if setting.Response {
		if resp != nil {
			body, err := httputil.DumpResponse(resp, setting.ResponseBody)
			if err == nil {
				fields = append(fields, logger.ResponseDump(body))
			} else {
				errs = append(errs, err)
			}
		} else {
			fields = append(fields, logger.ResponseDump(nil))
		}
	} else {
		fields = append(fields, zap.String(logger.ResponseDumpKey, "hidden"))
	}

	fields = append(fields, zap.Errors("errors", errs))
	if len(fields) > 0 {
		l.Debug("http request sent", fields...)
	}

	return resp, err
}
