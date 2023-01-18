package logger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"libs/logger"
)

func TestBindFields(t *testing.T) {
	type args struct {
		ctx    context.Context
		fields []zap.Field
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"ok",
			args{
				context.TODO(),
				[]zap.Field{logger.ProspectID("1"), logger.RequestDump([]byte("dump here")), logger.ApplicationID("app_id")},
			},
		},
	}

	testLog := logger.New("debug", "test")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logger.BindFields(tt.args.ctx, tt.args.fields...)
			bindFields, ok := got.Value(logger.BindFieldsKey).([]zap.Field)
			assert.True(t, ok)
			assert.Equal(t, len(tt.args.fields), len(bindFields))

			l := logger.WithContext(testLog, got)
			l.Info("tested")
		})
	}
}
