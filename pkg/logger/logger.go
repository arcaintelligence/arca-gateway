package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// Level representa o nível de log
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Entry representa uma entrada de log
type Entry struct {
	Level     string                 `json:"level"`
	Timestamp string                 `json:"timestamp"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
}

// Logger logger estruturado
type Logger struct {
	mu        sync.Mutex
	output    io.Writer
	level     Level
	fields    map[string]interface{}
	addCaller bool
}

// Config configuração do logger
type Config struct {
	Level     Level
	Output    io.Writer
	AddCaller bool
}

// New cria um novo logger
func New(cfg Config) *Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	return &Logger{
		output:    cfg.Output,
		level:     cfg.Level,
		fields:    make(map[string]interface{}),
		addCaller: cfg.AddCaller,
	}
}

// Default retorna um logger padrão
func Default() *Logger {
	return New(Config{
		Level:     InfoLevel,
		Output:    os.Stdout,
		AddCaller: true,
	})
}

// WithField adiciona um campo ao logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		output:    l.output,
		level:     l.level,
		fields:    make(map[string]interface{}),
		addCaller: l.addCaller,
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return newLogger
}

// WithFields adiciona múltiplos campos ao logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		output:    l.output,
		level:     l.level,
		fields:    make(map[string]interface{}),
		addCaller: l.addCaller,
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// log escreve uma entrada de log
func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	entry := Entry{
		Level:     level.String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Message:   fmt.Sprintf(msg, args...),
	}

	if len(l.fields) > 0 {
		entry.Fields = l.fields
	}

	if l.addCaller {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			entry.Caller = fmt.Sprintf("%s:%d", file, line)
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, _ := json.Marshal(entry)
	fmt.Fprintln(l.output, string(data))

	if level == FatalLevel {
		os.Exit(1)
	}
}

// Debug log de debug
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(DebugLevel, msg, args...)
}

// Info log de info
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(InfoLevel, msg, args...)
}

// Warn log de warning
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(WarnLevel, msg, args...)
}

// Error log de erro
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(ErrorLevel, msg, args...)
}

// Fatal log fatal (encerra o programa)
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(FatalLevel, msg, args...)
}

// =============================================================================
// GLOBAL LOGGER
// =============================================================================

var defaultLogger = Default()

// SetDefault define o logger padrão global
func SetDefault(l *Logger) {
	defaultLogger = l
}

// Debug log de debug global
func Debug(msg string, args ...interface{}) {
	defaultLogger.Debug(msg, args...)
}

// Info log de info global
func Info(msg string, args ...interface{}) {
	defaultLogger.Info(msg, args...)
}

// Warn log de warning global
func Warn(msg string, args ...interface{}) {
	defaultLogger.Warn(msg, args...)
}

// Error log de erro global
func Error(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
}

// Fatal log fatal global
func Fatal(msg string, args ...interface{}) {
	defaultLogger.Fatal(msg, args...)
}

// WithField adiciona campo ao logger global
func WithField(key string, value interface{}) *Logger {
	return defaultLogger.WithField(key, value)
}

// WithFields adiciona campos ao logger global
func WithFields(fields map[string]interface{}) *Logger {
	return defaultLogger.WithFields(fields)
}
