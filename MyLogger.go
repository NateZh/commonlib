package commonlib

import (
	"fmt"
	"github.com/astaxie/beego"
	"runtime"
)

var (
	Log *MyLogger //提供公用的日志方式
)

func init() {
	Log = new(MyLogger)
}

type MyLogger struct {
}

func (log *MyLogger) Error(arg0 ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	beego.Error("(文件:", file, ",行:", line, ")", fmt.Sprint(arg0...))
}

func (log *MyLogger) Debug(arg0 ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	beego.Debug("(文件:", file, ",行:", line, ")", fmt.Sprint(arg0...))
}

func (log *MyLogger) Info(arg0 ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	beego.Info("(文件:", file, ",行:", line, ")", fmt.Sprint(arg0...))
}

func (log *MyLogger) Warn(arg0 ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	beego.Warn("(文件:", file, ",行:", line, ")", fmt.Sprint(arg0...))
}

func (log *MyLogger) Trace(arg0 ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	beego.Trace("(文件:", file, ",行:", line, ")", fmt.Sprint(arg0...))
}

func (log *MyLogger) DebugSchedule(scheduleId, childId string, arg0 ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	beego.Trace("[scheduleId:", scheduleId, ",childId:", childId, "]", "(文件:", file, ",行:", line, ")", fmt.Sprint(arg0...), "\n")
}
