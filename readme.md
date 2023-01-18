# Logger helpers

## About
Wrapper around uber/zap lib. Includes functionality for tracking process_id via context. Includes:
- get logger from context
- Enrich logger with context passed field
- List of conventional Fields
- Enriching context with fields to log (process_id/request_id/extra_fields...)
- Instrumentation 
  - ginlog - gin.Middleware with read from req context and log basic info about response
  - httplog - http.Transport with log req/resp functionality and passing logging context (process_id)


# Initialization
To init logger we need to call logger.New function. LogLevel would be converted to lowercase.
Namespace is printed as "logger":"namespace" in stdout and/or grafana interface

## On application bootstrap
```go
	l := logger.New("debug", "namespace")
	defer func() { _ = logger.Cleanup() }() // Cleanup function should be Run when the program finishes
```

## From Context
If you have only context without a particular instance of logger. 
This will return new logger instance retrieved from globalLogger. 
```go
l:= logger.FromContext(ctx,"namespace") // if you pass empty string for namespace it would use namespace of global logger
```

## With context
if You have logger passed as dependency and only want to set fields passed in context
```go
l:= logger.WithContext(uc.log, ctx)
```

# Predefined zap.Fields
In order to make generic dashboard in Grafana we introduced a list of predefined zap.Field implementations
for specific values

| Name                                 | json field in output    | Expected value                         |
|--------------------------------------|-------------------------|----------------------------------------|
| ApplicationID(string) zap.Field      | application_id          | DBO_application_service.Application.ID |
| Contact(type,value string) zap.Field | contact_id:[type,value] | onboarding_fl.Contacts.(type,value)    |
| IABSClientID(string) zap.Field       | iabs_client_id          | iabs.physical_clients.ID               |
| MQMessageID(string) zap.Field        | mq_message_id           | MessageBroker.Message.ID               |
| RequestDump([]byte) zap.Field        | request_dump            | request dump encoded in bytes          |
| ResponseDump([]byte) zap.Field       | response_dump           | response dump encoded in bytes         |
| ProductID(string) zap.Field          | product_id              | product_catalog.products.ID            |
| ProspectID(string) zap.Field         | prospect_id             | onboarding_fl.Prospect.ID              |
| Stack() zap.Field                    | stack                   | Stacktrace of current goroutine        |

# Tracing
Usually we need to see what logs were printed for some action. For each of consequent calls in
one process we should add some identifier parameters. These parameters are passed to context.
In order for it to work you should pass context carefully through all layers.
### List of conventional parameters

| Name       | Description                                                                                                                                                                                              | 
|------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| process_id | ID of current process. It is passed to other services in further integrations by instrumentation drivers<br/>Commonly known as trace_id, as we are planning to make Jeager tracing this naming is chosen |
| request_id | ID of current http request. It is bounded to one specific request inside particular microservice                                                                                                         |
| message_id | ID of current async Message (AMQP etc). It is bounded to one specific message inside particular microservice                                                                                             |

## Binding log values to context
In order to bind values to context you can call any of three functions below

### BindProcessID(ctx context.Context, processID string) context.Context
Binds process_id that will be logged with each log created with this context.
Usually you won't need to call this func for HTTP(gin/http.Client)
as it is done internally by corresponding Instrumentation.

### BindRequestID(ctx context.Context, requestID string) context.Context
Binds request_id that will be logged with each log created with this context.
Usually you won't need to call this func for HTTP(gin/http.Client)
as it is done internally by corresponding Instrumentation.

### BindFields(ctx context.Context, fields ...zap.Field) context.Context
Binds all the fields provided to the context. All loggers initialized with the <b>returned</b> context
will print the fields. Can be called multiple times, internally appends existing fields to already declared

# Instrumentation
Provides ready to use middlewares for passing context to different microservices/systems

### Init GIN with Logger middleware 
This middleware will try to find process_id and request_id inside request headers
and put it into request.Context(). If no headers are found new UUID values are created and passed to context
```go
import "gitlab.hamkorbank.uz/libs/logger/instrumentation/ginlog"

r := gin.New()
r.Use(ginlog.Log(),ginlog.RecoveryWithZap(true) ) 
// if you have swagger and you want to skip it from logs use
r.Use(ginlog.LogExcept([]string{"/swagger/*any"}))
```


### Client for HTTP Transport
HTTP Transport is implementation of http.RoundTripper interface for outgoing requests and received responses
> For httplog.Transport to have access to your process context you should create request with NewRequestWithContext

Transport also provides skip options for request/requestBody/response/responseBody 
```go
var httpClient = &http.Client{
	Transport: httplog.New(http.DefaultTransport,
		httplog.NoContext(http.MethodPost, "/no-logger-header-would-be-sent")
		httplog.SkipReqBody(http.MethodPost, "/skipped-path1", "/skipped-path2"),
		httplog.SkipRequest(http.MethodPost, "/totaly-skipped-post-request"),
		httplog.SkipResponse(http.MethodGet, "/do-not-log-response-at-all"),
		httplog.SkipReqBody(http.MethodPost, "/do-not-log-response-body"),
	),
	Timeout: 20 * time.Second,
}
```
