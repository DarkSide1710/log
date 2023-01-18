package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"libs/logger"
	"libs/logger/instrumentation/ginlog"
	"libs/logger/instrumentation/httplog"
)

var httpClient = &http.Client{
	Transport: httplog.New(http.DefaultTransport,
		httplog.SkipReqBody(http.MethodPost, "/skipped-path1", "/skipped-path2"),
		httplog.SkipRequest(http.MethodPost, "/totaly-skipped-post-request"),
		httplog.SkipResponse(http.MethodGet, "/do-not-log-response-at-all"),
		httplog.SkipReqBody(http.MethodPost, "/do-not-log-response-body"),
	),
	Timeout: 20 * time.Second,
}

func main() {
	l := logger.New("debug", "example")
	defer func() { _ = logger.Cleanup() }()
	l.Info("Initialized")
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(ginlog.Log())
	r.Use(ginlog.RecoveryWithZap(true))
	r.GET("/", func(c *gin.Context) {
		l := logger.FromCtx(c.Request.Context(), "rest")
		l.Info("test-Info", zap.String("something", "in the way"))
		l.Debug("test-Debug", zap.String("something", "in the way"))
		l.Warn("test-Warn", zap.String("something", "in the way"))
		l.Error("test-Error", zap.String("something", "in the way"))
		c.JSON(http.StatusOK, map[string]interface{}{"logged": true})
	})
	r.GET("/panic", func(c *gin.Context) {
		panic("An unexpected error happen!")
	})

	go func() {
		ticker := time.Tick(time.Second * 3)

		i := 0
		select {
		case <-ticker:
			i++
			exec(logger.BindProcessID(context.Background(), fmt.Sprintf("process_%d", i)))
		}
	}()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}

func exec(ctx context.Context) {
	l := logger.FromCtx(ctx, "exec")
	l.Info("test-exec-Info", zap.String("Mmm", "Mmm"))
	l.Debug("test-exec-Debug", zap.String("Mmm", "Mmm"))
	l.Warn("test-exec-Warn", zap.String("Mmm", "Mmm"))
	l.Error("test-exec-Error", zap.String("Mmm", "Mmm"))
	body := map[string]interface{}{"test_data": 2, "logging": true, "Mmm": "Mmm", "group": "Nirvana"}
	var raw bytes.Buffer
	encoder := json.NewEncoder(&raw)
	if err := encoder.Encode(body); err != nil {
		l.Error("encoder.Encode", zap.Error(err))
		return
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:8081/totaly-skipped-post-request", &raw)
	if err != nil {
		l.Error("NewRequestWithContext", zap.Error(err))
		return
	}
	if _, err = httpClient.Do(r); err != nil {
		l.Error("httpClient.Do(r)")
		return
	}
}
