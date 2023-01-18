package ginlog

import (
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"libs/logger"
)

// Log returns a gin.HandlerFunc (middleware) that logs requests using uber-go/zap.
//
// Requests with errors are logged using zap.Error().
// Requests without errors are logged using zap.Info().
//
// It receives:
//  1. A time package format string (e.g. time.RFC3339).
//  2. A boolean stating whether to use UTC time zone or local.
func Log() gin.HandlerFunc {
	return LogExcept(nil)
}

// LogExcept returns a gin.HandlerFunc with logging except routes in skipPath
func LogExcept(skipPath []string) gin.HandlerFunc {
	skipPaths := make(map[string]bool, len(skipPath))
	for _, path := range skipPath {
		skipPaths[path] = true
	}

	logFunc := func(l logger.Logger, httpCode int) func(string, ...zap.Field) {
		switch {
		case httpCode >= http.StatusBadRequest:
			return l.Warn
		case httpCode >= http.StatusMultipleChoices:
			return l.Info
		default:
			return l.Debug
		}
	}

	return func(c *gin.Context) {
		start := time.Now()
		processID := c.GetHeader(logger.HTTPHeaderProcessID)
		if processID == "" {
			processID = uuid.NewString()
		}
		ctx := logger.BindProcessID(c.Request.Context(), processID)

		requestID := c.GetHeader(logger.HTTPHeaderRequestID)
		if requestID == "" {
			requestID = uuid.NewString()
		}
		ctx = logger.BindRequestID(ctx, requestID)

		c.Set(string(logger.ProcessIDKey), processID)
		c.Set(string(logger.RequestIDKey), requestID)
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if _, ok := skipPaths[path]; !ok {
			end := time.Now()
			latency := end.Sub(start)
			l := logger.FromCtx(ctx, "gin")
			if len(c.Errors) > 0 {
				// Append error field if this is an erroneous request.
				for _, e := range c.Errors.Errors() {
					l.Error(e)
				}
			} else {
				httpCode := c.Writer.Status()
				fields := []zapcore.Field{
					zap.Int("http_code", httpCode),
					zap.String("method", c.Request.Method),
					zap.String("path", path),
					zap.String("query", query),
					zap.String("ip", c.ClientIP()),
					zap.String("user-agent", c.Request.UserAgent()),
					zap.Duration("latency", latency),
				}
				logFunc(l, httpCode)(path, fields...)
			}
		}
	}
}

// RecoveryWithZap returns a gin.HandlerFunc (middleware)
// that recovers from any panics and logs requests using uber-go/zap.
// All errors are logged using zap.Error().
// stack means whether output the stack info.
// The stack info is easy to find where the error occurs but the stack info is too large.
func RecoveryWithZap(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)

				l := logger.FromCtx(c.Request.Context(), "gin")
				if brokenPipe {
					l.Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
					return
				}

				if stack {
					l.Error("[Recovery from panic]",
						zap.Time("time", time.Now()),
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					l.Error("[Recovery from panic]",
						zap.Time("time", time.Now()),
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
