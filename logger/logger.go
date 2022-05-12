package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[37m"
	logFormat   = "2006/01/02 15:04:05"
)

var (
	l  *logger
	ow *operationWriter
	jw *jobWriter
)

type logWriter struct {
	io.Writer
	format string
}

func (w logWriter) Write(b []byte) (n int, err error) {
	return w.Writer.Write(append([]byte(time.Now().Format(w.format)), b...))
}

type operationWriter struct {
	io.Writer

	mux    sync.Mutex
	format string
}

func (ow *operationWriter) Write(b []byte) (n int, err error) {
	ow.mux.Lock()
	defer ow.mux.Unlock()

	l.op.Print(string(b))
	return len(b), nil
}

type jobWriter struct {
	io.Writer

	mux    sync.Mutex
	format string
}

func (jw *jobWriter) Write(b []byte) (n int, err error) {
	jw.mux.Lock()
	defer jw.mux.Unlock()

	l.job.Print(string(b))
	return len(b), nil
}

// NewInstance creates a new singleton logging instance.
func NewInstance(debug bool) {
	ow = &operationWriter{
		format: logFormat,
	}

	jw = &jobWriter{
		format: logFormat,
	}

	l = &logger{
		debugEnabled: debug,
		debug: log.New(&logWriter{
			Writer: os.Stdout,
			format: logFormat,
		}, fmt.Sprintf(" [%sdbg%s] ", colorGray, colorReset), 0),
		op: log.New(&logWriter{
			Writer: os.Stdout,
			format: logFormat,
		}, fmt.Sprintf(" [%sopr%s] ", colorCyan, colorReset), 0),
		job: log.New(&logWriter{
			Writer: os.Stdout,
			format: logFormat,
		}, fmt.Sprintf(" [%sjob%s] ", colorPurple, colorReset), 0),
		info: log.New(&logWriter{
			Writer: os.Stdout,
			format: logFormat,
		}, " [inf] ", 0),
		warn: log.New(&logWriter{
			Writer: os.Stdout,
			format: logFormat,
		}, fmt.Sprintf(" [%swrn%s] ", colorYellow, colorReset), 0),
		error: log.New(&logWriter{
			Writer: os.Stderr,
			format: logFormat,
		}, fmt.Sprintf(" [%serr%s] ", colorRed, colorReset), 0),
	}
}

type logger struct {
	debugEnabled bool
	debug        *log.Logger
	op           *log.Logger
	job          *log.Logger
	info         *log.Logger
	warn         *log.Logger
	error        *log.Logger
}

func OperationWriter() io.Writer {
	return ow
}

func JobWriter() io.Writer {
	return jw
}

func Debug(msg string) {
	if !l.debugEnabled {
		return
	}
	l.debug.Println(msg)
}

func Debugf(format string, v ...any) {
	if !l.debugEnabled {
		return
	}
	l.debug.Printf(format, v...)
}

func Op(msg string) {
	l.op.Println(msg)
}

func Opf(format string, v ...any) {
	l.op.Printf(format, v...)
}

func Job(msg string) {
	l.job.Println(msg)
}

func Jobf(format string, v ...any) {
	l.job.Printf(format, v...)
}

func Info(msg string) {
	l.info.Println(msg)
}

func Infof(format string, v ...any) {
	l.info.Printf(format, v...)
}

func Warn(msg string) {
	l.warn.Println(msg)
}

func Warnf(format string, v ...any) {
	l.warn.Printf(format, v...)
}

func Error(v ...any) {
	l.error.Println(v...)
}

func Errorf(format string, v ...any) {
	l.error.Printf(format, v...)
}

func Fatal(v ...any) {
	l.error.Fatal(v...)
}

func Fatalf(format string, v ...any) {
	l.error.Fatalf(format, v)
}
