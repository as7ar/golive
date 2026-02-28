package logger

import (
	"fmt"
	"time"
)

func log(color string, level string, content ...interface{}) {
	fmt.Printf("%s %s [%s] %s%s\n",
		color,
		time.Now().Format("2006-01-02 15:04:05"),
		level,
		fmt.Sprint(content...),
		"\033[0m",
	)
}

func Info(content ...interface{}) {
	log("\033[37m", "INFO", content...)
}

func Err(content ...interface{}) {
	log("\033[31m", "ERROR", content...)
}

func Debug(content ...interface{}) {
	log("\033[33m", "DEBUG", content...)
}
