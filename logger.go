package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zapLogger *zap.Logger

// Logger ...
type Logger interface {
	Debug(msg string, fields ...zapcore.Field)
	Info(msg string, fields ...zapcore.Field)
	Warn(msg string, fields ...zapcore.Field)
	Error(msg string, fields ...zapcore.Field)
	Fatal(msg string, fields ...zapcore.Field)
}

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(time.RFC3339))
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func init() {
	zapLogger = zap.NewNop()
}

func New(level, namespace string, options ...zap.Option) *zap.Logger {
	globalLevel := parseLevel(level)
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= globalLevel && lvl < zapcore.ErrorLevel
	})
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	consoleInfos := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = customTimeEncoder
	consoleEncoder := zapcore.NewJSONEncoder(encoderCfg)
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleInfos, lowPriority),
	)

	if len(options) == 0 {
		options = []zap.Option{zap.WithCaller(true), zap.AddCallerSkip(1)}
	}

	zapLogger = zap.New(core, options...).Named(namespace)
	zap.RedirectStdLog(zapLogger)

	return zapLogger
}

// BindRequestID returns a context which knows its request ID
func BindRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithReqID - returns a context which knows its session ID
// Deprecated: use BindRequestID
var WithReqID = BindRequestID

func BindFields(ctx context.Context, fields ...zap.Field) context.Context {
	if existing, ok := ctx.Value(BindFieldsKey).([]zap.Field); ok && existing != nil {
		return context.WithValue(ctx, BindFieldsKey, append(existing, fields...))
	}

	return context.WithValue(ctx, BindFieldsKey, fields)
}

// BindProcessID returns a context which knows its process ID
func BindProcessID(ctx context.Context, processID string) context.Context {
	return context.WithValue(ctx, ProcessIDKey, processID)
}

// WithProcessID - returns a context which knows its session ID
// Deprecated: use BindProcessID
var WithProcessID = BindProcessID

// GetProcessID Retrieves process_id from context. Handy for passing process_id to external request identifiers
// e.g. WSO2ESB requestID, amqp.Publishing.Header
func GetProcessID(ctx context.Context) string {
	if processID, ok := ctx.Value(ProcessIDKey).(string); ok && processID != "" {
		return processID
	}
	return uuid.NewString()
}

// FromCtx returns a zap logger with as much context as possible. namespace is added to global namespace.
// {"logger": "globalNamespace.namespace"}
func FromCtx(ctx context.Context, namespace string) *zap.Logger {
	newLogger := zapLogger
	if ctx != nil {
		if ctxReqID, ok := ctx.Value(RequestIDKey).(string); ok {
			newLogger = newLogger.With(zap.String("request_id", ctxReqID))
		}
		if processID, ok := ctx.Value(ProcessIDKey).(string); ok {
			newLogger = newLogger.With(zap.String("process_id", processID))
		}
		if bindFields, ok := ctx.Value(BindFieldsKey).([]zap.Field); ok {
			newLogger = newLogger.With(bindFields...)
		}
	}
	if namespace != "" {
		newLogger = newLogger.Named(namespace)
	}
	return newLogger
}

func WithContext(l Logger, ctx context.Context) *zap.Logger {
	newLogger, ok := l.(*zap.Logger)
	if !ok {
		newLogger = zapLogger
	}
	if ctx != nil {
		if ctxReqID, ok := ctx.Value(RequestIDKey).(string); ok {
			newLogger = newLogger.With(zap.String("request_id", ctxReqID))
		}
		if processID, ok := ctx.Value(ProcessIDKey).(string); ok {
			newLogger = newLogger.With(zap.String("process_id", processID))
		}
		if bindFields, ok := ctx.Value(BindFieldsKey).([]zap.Field); ok {
			newLogger = newLogger.With(bindFields...)
		}
	}

	return newLogger
}

func Cleanup() error {
	if zapLogger != nil {
		return zapLogger.Sync()
	}
	return nil
}
