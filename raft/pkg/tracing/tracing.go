package tracing

import (
	"github.com/opentracing/opentracing-go"

	"github.com/uber/jaeger-client-go/config"
	zwrap "github.com/uber/jaeger-client-go/log/zap"
	"go.uber.org/zap"

	"io"
)

// Init creates a new instance of Jaeger tracer.
func Init(serviceName string, logger *zap.Logger) (opentracing.Tracer, io.Closer) {

	cfg, err := config.FromEnv()
	if err != nil {
		logger.Fatal("cannot parse Jaeger env vars", zap.Error(err))
	}
	cfg.ServiceName = serviceName
	cfg.Sampler.Type = "const"
	cfg.Sampler.Param = 1


	jLogger := zwrap.NewLogger(logger)

	tracer, closer, _ := cfg.NewTracer(
		config.Logger(jLogger),
	)
	if err != nil {
		logger.Fatal("cannot initialize Jaeger Tracer", zap.Error(err))
	}
	return tracer, closer
}
