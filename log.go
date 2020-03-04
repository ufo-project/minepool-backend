package main

import (
	"build"
	"fmt"
	"io"
	"log"
	"os"
)

var (
	Debug   Logger
	Info    *log.Logger
	Warning *log.Logger
	Alert   *log.Logger

	ShareLog *log.Logger
	BlockLog *log.Logger
)

const (
	DEBUG_LEVEL   = 7
	INFO_LEVEL    = 6
	WARNING_LEVEL = 4
	ALERT_LEVEL   = 1
)

func InitLog(infoFile, errorFile, shareFile, blockFile string) {
	log.Println("infoFile:", infoFile)
	log.Println("errorFile:", errorFile)
	log.Println("shareFile:", shareFile)
	log.Println("blockFile:", blockFile)
	infoFd, err := os.OpenFile(infoFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open error log file:", err)
	}

	errorFd, err := os.OpenFile(errorFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open error log file:", err)
	}

	shareFd, err := os.OpenFile(shareFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open error log file:", err)
	}

	blockFd, err := os.OpenFile(blockFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open error log file:", err)
	}

	Debug.l = log.New(io.MultiWriter(os.Stdout, infoFd), "[DEBUG] ", log.Ldate|log.Lmicroseconds|log.Lshortfile)
	Info = log.New(io.MultiWriter(os.Stdout, infoFd), "[INFO] ", log.Ldate|log.Lmicroseconds|log.Lshortfile)
	Warning = log.New(io.MultiWriter(os.Stdout, infoFd, errorFd), "[WARN] ", log.Ldate|log.Lmicroseconds|log.Lshortfile)
	Alert = log.New(io.MultiWriter(os.Stdout, infoFd, errorFd), "[ALERT] ", log.Ldate|log.Lmicroseconds|log.Lshortfile)

	ShareLog = log.New(io.MultiWriter(shareFd), "", log.Ldate|log.Lmicroseconds)
	BlockLog = log.New(io.MultiWriter(blockFd, os.Stdout), "", log.Ldate|log.Lmicroseconds)
}

type Logger struct {
	l *log.Logger
}

func (l *Logger) Print(v ...interface{}) {
	if !build.DEBUG {
		return
	}
	l.l.Output(2, fmt.Sprint(v...))
}

func (l *Logger) Println(v ...interface{}) {
	if !build.DEBUG {
		return
	}
	l.l.Output(2, fmt.Sprintln(v...))
}

func (l *Logger) Printf(format string, v ...interface{}) {
	if !build.DEBUG {
		return
	}
	l.l.Output(2, fmt.Sprintf(format, v...))
}

func (l *Logger) Fatal(v ...interface{}) {
	if !build.DEBUG {
		return
	}
	l.l.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	if !build.DEBUG {
		return
	}
	l.l.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *Logger) Fatalln(v ...interface{}) {
	if !build.DEBUG {
		return
	}
	l.l.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}

func (l *Logger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	if !build.DEBUG {
		return
	}
	l.l.Output(2, s)
	panic(s)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	if !build.DEBUG {
		return
	}
	l.l.Output(2, s)
	panic(s)
}

func (l *Logger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	if !build.DEBUG {
		return
	}
	l.l.Output(2, s)
	panic(s)
}
