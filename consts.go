package logger

type correlationIDCtxKey string

const (
	RequestIDKey  correlationIDCtxKey = "loggerRequestIDKey"
	BindFieldsKey correlationIDCtxKey = "loggerBindFields"
	ProcessIDKey  correlationIDCtxKey = "loggerProcessIDKey"
)

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

const (
	HTTPHeaderRequestID = "x-log-request-id"
	HTTPHeaderProcessID = "x-log-process-id"
)
