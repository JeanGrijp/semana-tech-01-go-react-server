// Package logger provides a structured logger for the application.
package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

/* -------------------------------------------------------------------------- */
/*                        1.  Request-scoped values                           */
/* -------------------------------------------------------------------------- */

type contextKey string

const (
	requestIDKey           contextKey = "request_id"
	clientIPKey            contextKey = "client_ip"
	userAgentKey           contextKey = "user_agent"
	methodKey              contextKey = "method"
	pathKey                contextKey = "path"
	queryKey               contextKey = "query"
	refererKey             contextKey = "referer"
	hostKey                contextKey = "host"
	latencyKey             contextKey = "latency"
	statusCodeKey          contextKey = "status_code"
	authenticatedUserIDKey contextKey = "authenticated_user_id"
)

/* --- setters & getters ---------------------------------------------------- */

func NewRequestID() string { return uuid.New().String() }

func InjectRequestID(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestIDKey, NewRequestID())
}
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func InjectClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey, ip)
}
func ClientIPFromContext(ctx context.Context) string {
	if ip, ok := ctx.Value(clientIPKey).(string); ok {
		return ip
	}
	return ""
}

func InjectUserAgent(ctx context.Context, ua string) context.Context {
	return context.WithValue(ctx, userAgentKey, ua)
}
func UserAgentFromContext(ctx context.Context) string {
	if ua, ok := ctx.Value(userAgentKey).(string); ok {
		return ua
	}
	return ""
}

func InjectMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, methodKey, method)
}
func MethodFromContext(ctx context.Context) string {
	if m, ok := ctx.Value(methodKey).(string); ok {
		return m
	}
	return ""
}

func InjectPath(ctx context.Context, p string) context.Context {
	return context.WithValue(ctx, pathKey, p)
}
func PathFromContext(ctx context.Context) string {
	if p, ok := ctx.Value(pathKey).(string); ok {
		return p
	}
	return ""
}

func InjectQuery(ctx context.Context, q string) context.Context {
	return context.WithValue(ctx, queryKey, q)
}
func QueryFromContext(ctx context.Context) string {
	if q, ok := ctx.Value(queryKey).(string); ok {
		return q
	}
	return ""
}

func InjectReferer(ctx context.Context, ref string) context.Context {
	return context.WithValue(ctx, refererKey, ref)
}
func RefererFromContext(ctx context.Context) string {
	if r, ok := ctx.Value(refererKey).(string); ok {
		return r
	}
	return ""
}

func InjectHost(ctx context.Context, h string) context.Context {
	return context.WithValue(ctx, hostKey, h)
}
func HostFromContext(ctx context.Context) string {
	if h, ok := ctx.Value(hostKey).(string); ok {
		return h
	}
	return ""
}

func InjectLatency(ctx context.Context, d time.Duration) context.Context {
	return context.WithValue(ctx, latencyKey, d)
}
func LatencyFromContext(ctx context.Context) time.Duration {
	if d, ok := ctx.Value(latencyKey).(time.Duration); ok {
		return d
	}
	return 0
}

func InjectStatusCode(ctx context.Context, code int) context.Context {
	return context.WithValue(ctx, statusCodeKey, code)
}
func StatusCodeFromContext(ctx context.Context) int {
	if s, ok := ctx.Value(statusCodeKey).(int); ok {
		return s
	}
	return 0
}

func InjectAuthenticatedUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, authenticatedUserIDKey, id)
}
func AuthenticatedUserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(authenticatedUserIDKey).(string); ok {
		return id
	}
	return ""
}

/* -------------------------------------------------------------------------- */
/*                        2.  Zap logger wrapper                              */
/* -------------------------------------------------------------------------- */

type Logger interface {
	Info(ctx context.Context, msg string, fields ...any)
	Debug(ctx context.Context, msg string, fields ...any)
	Warn(ctx context.Context, msg string, fields ...any)
	Error(ctx context.Context, msg string, fields ...any)
	Fatal(ctx context.Context, msg string, fields ...any)
}

type ZapLogger struct {
	logger *zap.SugaredLogger
}

/* ---------- encoders & cores --------------------------------------------- */

var consoleEncoder = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
	TimeKey:     "T",
	LevelKey:    "L",
	MessageKey:  "M",
	EncodeTime:  zapcore.TimeEncoderOfLayout("15:04:05"),
	EncodeLevel: zapcore.CapitalColorLevelEncoder,
})

var jsonEncoder = zapcore.NewJSONEncoder(zapcore.EncoderConfig{
	TimeKey:      "timestamp",
	LevelKey:     "level",
	MessageKey:   "message",
	CallerKey:    "caller",
	EncodeTime:   zapcore.ISO8601TimeEncoder,
	EncodeLevel:  zapcore.CapitalLevelEncoder,
	EncodeCaller: zapcore.ShortCallerEncoder,
})

func NewZapLogger() *ZapLogger {
	// choose level from env
	level := zap.InfoLevel
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = zap.DebugLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	}

	// rotating file writer
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "./internal/logger/logs/api.log",
		MaxSize:    10, // MB
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   true,
	})

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
		zapcore.NewCore(jsonEncoder, fileWriter, level),
	)

	zapLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	).Sugar()

	return &ZapLogger{logger: zapLogger}
}

/* ---------- helper: common context fields -------------------------------- */

func baseFields(ctx context.Context) []any {
	return []any{
		"request_id", RequestIDFromContext(ctx),
		"client_ip", ClientIPFromContext(ctx),
		"user_agent", UserAgentFromContext(ctx),
		"method", MethodFromContext(ctx),
		"path", PathFromContext(ctx),
		"query", QueryFromContext(ctx),
		"referer", RefererFromContext(ctx),
		"host", HostFromContext(ctx),
		"latency", LatencyFromContext(ctx),
		"status_code", StatusCodeFromContext(ctx),
		"user_id", AuthenticatedUserIDFromContext(ctx),
	}
}

/* ---------- API implementation ------------------------------------------- */

func (z *ZapLogger) Info(ctx context.Context, msg string, fields ...any) {
	z.logger.Infow(msg, append(baseFields(ctx), fields...)...)
}
func (z *ZapLogger) Debug(ctx context.Context, msg string, fields ...any) {
	z.logger.Debugw(msg, append(baseFields(ctx), fields...)...)
}
func (z *ZapLogger) Warn(ctx context.Context, msg string, fields ...any) {
	z.logger.Warnw(msg, append(baseFields(ctx), fields...)...)
}
func (z *ZapLogger) Error(ctx context.Context, msg string, fields ...any) {
	z.logger.Errorw(msg, append(baseFields(ctx), fields...)...)
}
func (z *ZapLogger) Fatal(ctx context.Context, msg string, fields ...any) {
	z.logger.Fatalw(msg, append(baseFields(ctx), fields...)...)
}

/* ---------- default instance exported ------------------------------------ */

var Default Logger = NewZapLogger()
