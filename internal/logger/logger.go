package logger

import (
	"log"
	"os"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	level Level
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
}

func New(levelStr string) *Logger {
	level := parseLevel(levelStr)

	return &Logger{
		level: level,
		debug: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		info:  log.New(os.Stdout, "[INFO]  ", log.LstdFlags),
		warn:  log.New(os.Stdout, "[WARN]  ", log.LstdFlags),
		error: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
	}
}

func parseLevel(levelStr string) Level {
	switch levelStr {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= LevelDebug {
		l.debug.Printf(format, v...)
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= LevelInfo {
		l.info.Printf(format, v...)
	}
}

func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= LevelWarn {
		l.warn.Printf(format, v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= LevelError {
		l.error.Printf(format, v...)
	}
}

func (l *Logger) Fatal(format string, v ...interface{}) {
	l.error.Printf(format, v...)
	os.Exit(1)
}
