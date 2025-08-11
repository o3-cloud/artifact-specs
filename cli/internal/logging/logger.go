package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Level int

const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace // Extra verbose level for -vv and higher
)

var levelNames = map[Level]string{
	LevelError: "ERROR",
	LevelWarn:  "WARN",
	LevelInfo:  "INFO",
	LevelDebug: "DEBUG",
	LevelTrace: "TRACE",
}

type Logger struct {
	level    Level
	jsonMode bool
	output   io.Writer
}

var defaultLogger *Logger

func init() {
	defaultLogger = &Logger{
		level:    LevelInfo,
		jsonMode: false,
		output:   os.Stderr,
	}
}

func SetLevel(level Level) {
	defaultLogger.level = level
}

func SetJSONMode(enabled bool) {
	defaultLogger.jsonMode = enabled
}

func SetQuiet() {
	defaultLogger.level = LevelError
}

func SetVerbose(count int) {
	switch count {
	case 1:
		defaultLogger.level = LevelDebug
	case 2:
		defaultLogger.level = LevelTrace
	default:
		if count > 2 {
			defaultLogger.level = LevelTrace
		} else {
			defaultLogger.level = LevelInfo
		}
	}
}

func (l *Logger) log(level Level, msg string, fields map[string]interface{}) {
	if level > l.level {
		return
	}
	
	if l.jsonMode {
		entry := map[string]interface{}{
			"level":   levelNames[level],
			"message": msg,
		}
		for k, v := range fields {
			entry[k] = v
		}
		
		data, _ := json.Marshal(entry)
		fmt.Fprintln(l.output, string(data))
	} else {
		prefix := fmt.Sprintf("[%s] ", levelNames[level])
		if fields != nil && len(fields) > 0 {
			fieldStr := ""
			for k, v := range fields {
				fieldStr += fmt.Sprintf(" %s=%v", k, v)
			}
			fmt.Fprintf(l.output, "%s%s%s\n", prefix, msg, fieldStr)
		} else {
			fmt.Fprintf(l.output, "%s%s\n", prefix, msg)
		}
	}
}

func Error(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	defaultLogger.log(LevelError, msg, f)
}

func Warn(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	defaultLogger.log(LevelWarn, msg, f)
}

func Info(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	defaultLogger.log(LevelInfo, msg, f)
}

func Debug(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	defaultLogger.log(LevelDebug, msg, f)
}

func Trace(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	defaultLogger.log(LevelTrace, msg, f)
}

func Fatal(msg string, fields ...map[string]interface{}) {
	Error(msg, fields...)
	os.Exit(1)
}