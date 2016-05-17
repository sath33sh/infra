// This package provides a leveled logging facility.
package log

import (
	"fmt"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"io/ioutil"
	stdlog "log"
	"os"
	"sync"
)

// Following levels are supported:
// FATAL - Unrecoverable error which causes the caller to panic.
// ERROR - Error log.
// DEBUG - Debug log.
const (
	FATAL = iota
	ERROR
	DEBUG
)

var (
	level       int = ERROR
	debugEnable     = map[string]bool{}
	lock        sync.Mutex
	lj          = lumberjack.Logger{
		MaxSize:    20, // Megabytes.
		MaxBackups: 10,
		MaxAge:     30, // Days.
	}
	fatalLogger *stdlog.Logger
	errorLogger *stdlog.Logger
	debugLogger *stdlog.Logger
	infoLogger  *stdlog.Logger
)

func Fatalln(v ...interface{}) {
	if level >= FATAL {
		s := fmt.Sprintln(v...)
		fatalLogger.Output(2, s)
		panic(s)
	}
}

func Fatalf(format string, v ...interface{}) {
	if level >= FATAL {
		s := fmt.Sprintf(format, v...)
		fatalLogger.Output(2, s)
		panic(s)
	}
}

func Errorln(v ...interface{}) {
	if level >= ERROR {
		errorLogger.Output(2, fmt.Sprintln(v...))
	}
}

func Errorf(format string, v ...interface{}) {
	if level >= ERROR {
		errorLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func ErrorfOutput(calldepth int, format string, v ...interface{}) {
	if level >= ERROR {
		errorLogger.Output(calldepth, fmt.Sprintf(format, v...))
	}
}

func Debugln(module string, v ...interface{}) {
	if level >= DEBUG {
		if debugEnable[module] {
			debugLogger.Output(2, fmt.Sprintln(v...))
		}
	}
}

func Debugf(module, format string, v ...interface{}) {
	if level >= DEBUG {
		if debugEnable[module] {
			debugLogger.Output(2, fmt.Sprintf(format, v...))
		}
	}
}

func DebugfOutput(calldepth int, module, format string, v ...interface{}) {
	if level >= DEBUG {
		if debugEnable[module] {
			debugLogger.Output(calldepth, fmt.Sprintf(format, v...))
		}
	}
}

// NOTE: log.Info routines do not check for log level. They should be used sparingly in production code.
// It should be used only for informational purpose. Please do NOT use it for debug purposes.
func Infoln(v ...interface{}) {
	infoLogger.Output(2, fmt.Sprintln(v...))
}

func Infof(format string, v ...interface{}) {
	infoLogger.Output(2, fmt.Sprintf(format, v...))
}

func InfofOutput(calldepth int, format string, v ...interface{}) {
	infoLogger.Output(calldepth, fmt.Sprintf(format, v...))
}

func EnableDebug(module string) {
	lock.Lock()
	debugEnable[module] = true
	lock.Unlock()
}

func DisableDebug(module string) {
	lock.Lock()
	debugEnable[module] = false
	lock.Unlock()
}

func initLoggers(writer io.Writer) {
	fatalLogger = stdlog.New(writer, "FATAL: ", stdlog.Ldate|stdlog.Lmicroseconds|stdlog.Lshortfile)
	errorLogger = stdlog.New(writer, "ERROR: ", stdlog.Ldate|stdlog.Lmicroseconds|stdlog.Lshortfile)
	debugLogger = stdlog.New(writer, "DEBUG: ", stdlog.Ldate|stdlog.Lmicroseconds|stdlog.Lshortfile)
	infoLogger = stdlog.New(writer, "INFO: ", stdlog.Ldate|stdlog.Lmicroseconds|stdlog.Lshortfile)
}

func GetDebugLogger() *stdlog.Logger {
	return debugLogger
}

func Init(logFilePath string, logLevel string, stdout bool) {
	levelMap := map[string]int{
		"fatal": FATAL,
		"error": ERROR,
		"debug": DEBUG,
	}

	// Log level.
	levelStr := logLevel
	var ok bool
	if level, ok = levelMap[levelStr]; !ok {
		// Default to ERROR.
		level = ERROR
	}

	if logFilePath != "" {
		lj.Filename = logFilePath

		if stdout {
			// Log to file and stdout.
			initLoggers(io.MultiWriter(&lj, os.Stdout))
		} else {
			// Log to file.
			initLoggers(&lj)
		}

		Infof("Log level %d, file %s, stdout %v\n", level, logFilePath, stdout)
	} else if stdout {
		// Log to stdout only.
		initLoggers(os.Stdout)
	} else {
		initLoggers(ioutil.Discard)
	}
}
