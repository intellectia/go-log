package logger

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	zap *zap.Logger
}

type Config struct {
	InfoLogPath  string
	ErrorLogPath string
	Mode         string
}

var (
	logInstance *Logger
	once        sync.Once
)

func Init(config *Config) {
	once.Do(func() {
		logInstance = NewLogger(config)
	})
}

func zapErrorWithStack(err error) (msg zap.Field, stack zap.Field) {
	// Get the stack trace
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false) // false for all goroutines, true for current goroutine
	return zap.String("error", err.Error()), zap.String("stacktrace", string(buf[:n]))
}

func Info(msg string, tags ...zap.Field) {
	logInstance.Info(msg, tags...)
}

func Error(msg string, err error, tags ...zap.Field) {
	logInstance.Error(msg, err, tags...)
}

func Debug(msg string, tags ...zap.Field) {
	logInstance.Debug(msg, tags...)
}

func Warn(msg string, tags ...zap.Field) {
	logInstance.Warn(msg, tags...)
}

func Fatal(msg string, tags ...zap.Field) {
	logInstance.Fatal(msg, tags...)
}

// Formatted logging for Info level
func Infof(msg string, args ...interface{}) {
	logInstance.Info(fmt.Sprintf(msg, args...))
}

// Formatted logging for Error level
func Errorf(format string, args ...interface{}) {
	logInstance.Errorf(format, args...)
}

// Formatted logging for Debug level
func Debugf(msg string, args ...interface{}) {
	logInstance.Debug(fmt.Sprintf(msg, args...))
}

// Formatted logging for Warn level
func Warnf(msg string, args ...interface{}) {
	logInstance.Warn(fmt.Sprintf(msg, args...))
}

// Formatted logging for Fatal level
func Fatalf(msg string, args ...interface{}) {
	logInstance.Fatal(fmt.Sprintf(msg, args...))
}

func beijingTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	beijingLocation, _ := time.LoadLocation("Asia/Shanghai")
	enc.AppendString(t.In(beijingLocation).Format(time.RFC3339Nano))
}

func getEncoderConfig(config *Config) zapcore.EncoderConfig {
	logMode := config.Mode
	if logMode == "prod" {
		return zap.NewProductionEncoderConfig()
	}
	return zap.NewDevelopmentEncoderConfig()
}

func NewLogger(config *Config) *Logger {
	// Create a lumberjack logger (from "gopkg.in/natefinch/lumberjack.v2") for file rotation.
	infoLogWriter := &lumberjack.Logger{
		Filename:   config.InfoLogPath,
		MaxSize:    500, // megabytes after which new file is created
		MaxBackups: 3,   // number of backups
		MaxAge:     28,  //days
	}

	errorLogWriter := &lumberjack.Logger{
		Filename:   config.ErrorLogPath,
		MaxSize:    500,
		MaxBackups: 3,
		MaxAge:     28,
	}
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = beijingTimeEncoder

	// Create a zapcore.Core for each log level you need
	infoCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(infoLogWriter),
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.DebugLevel && lvl <= zapcore.WarnLevel
		}),
	)

	errorCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(errorLogWriter),
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		}),
	)

	consoleEncoderConfig := getEncoderConfig(config)

	// Create a zapcore.Core for stdout
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleEncoderConfig),
		zapcore.AddSync(zapcore.Lock(os.Stdout)),
		zapcore.DebugLevel, // or whichever minimum level you want to be printed to console
	)

	// Combine them together
	core := zapcore.NewTee(infoCore, errorCore, consoleCore)

	// Create a zap logger with the combined core
	zlog := zap.New(core)

	return &Logger{zap: zlog}
}

func (l *Logger) Info(msg string, tags ...zap.Field) {
	l.zap.Info(msg, tags...)
}

func (l *Logger) Error(msg string, err error, tags ...zap.Field) {
	errMsg, errStack := zapErrorWithStack(err)
	allFields := append(tags, zap.String("error", err.Error()), errMsg, errStack)
	l.zap.Error(msg, allFields...)
}

func (l *Logger) Debug(msg string, tags ...zap.Field) {
	l.zap.Debug(msg, tags...)
}

func (l *Logger) Warn(msg string, tags ...zap.Field) {
	l.zap.Warn(msg, tags...)
}

func (l *Logger) Fatal(msg string, tags ...zap.Field) {
	l.zap.Fatal(msg, tags...)
}

// Formatted logger methods

func (l *Logger) Infof(msg string, args ...interface{}) {
	l.zap.Info(fmt.Sprintf(msg, args...))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	var stackErr error
	for _, arg := range args {
		if e, ok := arg.(error); ok {
			stackErr = e
			break
		}
	}
	if stackErr != nil {
		errMsg, errStack := zapErrorWithStack(stackErr)
		l.zap.Error(msg, errMsg, errStack)
	} else {
		l.zap.Error(msg)
	}
}

func (l *Logger) Debugf(msg string, args ...interface{}) {
	l.zap.Debug(fmt.Sprintf(msg, args...))
}

func (l *Logger) Warnf(msg string, args ...interface{}) {
	l.zap.Warn(fmt.Sprintf(msg, args...))
}

func (l *Logger) Fatalf(msg string, args ...interface{}) {
	l.zap.Fatal(fmt.Sprintf(msg, args...))
}

func GetInstance() *Logger {
	return logInstance
}

// ... Implement similar functions for other log levels like Debug, Warn, Fatal ...

// Cleanup should be called to ensure all log messages are flushed
func Cleanup() {
	logInstance.zap.Sync()
}
