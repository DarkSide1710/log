package logger

import "go.uber.org/zap"

var (
	// Int - creates zap.Field for integer value. Preferably - zap.Int.
	// this constructor was made for easy migration from legacy pkg/logger.Int
	Int = zap.Int
	// String - creates zap.Field for string value. Preferably - zap.String.
	// this constructor was made for easy migration from legacy pkg/logger.String
	String = zap.String
	// Error - creates zap.Field for error value. Preferably - zap.Error.
	// this constructor was made for easy migration from legacy pkg/logger.Error
	Error = zap.Error
	// Bool - creates zap.Field for boolean value. Preferably - zap.Bool.
	// this constructor was made for easy migration from legacy pkg/logger.Bool
	Bool = zap.Bool
	// Any - creates zap.Field for Any type value. Preferably - Not to use it. If you actually do use zap.Any.
	// this constructor was made for easy migration from legacy pkg/logger.Any
	Any = zap.Any
)

const (
	ApplicationIDKey = "application_id"
	ContactIDKey     = "contact_id"
	IABSClientIDKey  = "iabs_client_id"
	MQMessageIDKey   = "mq_message_id"
	RequestDumpKey   = "request_dump"
	ResponseDumpKey  = "response_dump"
	ProductIDKey     = "product_id"
	ProspectIDKey    = "prospect_id"
	StackTraceKey    = "stack"
)

// ProspectID - use prospectID zap.Field for logging
// Will print prospect_id key to logs
func ProspectID(val string) zap.Field {
	return zap.String(ProspectIDKey, val)
}

// IABSClientID prints IabsClientID to logs
func IABSClientID(val string) zap.Field {
	return zap.String(IABSClientIDKey, val)
}

// ApplicationID generates zap.Field with conventional application_id key
// Application is Entity from dbo_application_service
func ApplicationID(val string) zap.Field {
	return zap.String(ApplicationIDKey, val)
}

// ProductID generates zap.Field with conventional product_id key
// Product is Entity from dbo_product_catalog_service
func ProductID(val string) zap.Field {
	return zap.String(ProductIDKey, val)
}

// Contact generates zap.Field with conventional naming
func Contact(id, value string) zap.Field {
	return zap.Strings(ContactIDKey, []string{id, value})
}

// Stack wraps zap.Stack function with conventional naming "stack"
func Stack() zap.Field {
	return zap.StackSkip(StackTraceKey, 1)
}

func RequestDump(body []byte) zap.Field {
	return zap.ByteString(RequestDumpKey, body)
}

func ResponseDump(body []byte) zap.Field {
	return zap.ByteString(ResponseDumpKey, body)
}

// MQMessageID generates zap.Field with mq_message_id
func MQMessageID(id string) zap.Field {
	return zap.String(MQMessageIDKey, id)
}
