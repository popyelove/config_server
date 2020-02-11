package stlog

import (
    "container/list"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "sync"
    "time"
)

type LogLevel int32
const (
    LogDebug LogLevel = 1
    LogWarn LogLevel = 2
    LogError LogLevel = 3
    LogInfo LogLevel = 4
    LogFatal LogLevel = 5
)

const normalLevel LogLevel = LogInfo



type Log interface {
    Debug(fmt string, v ...interface{})
    Warn(fmt string, v ...interface{})
    Error(fmt string, v ...interface{})
    Info(fmt string, v ...interface{})
    Fatal(fmt string, v ...interface{})
}


type STLogger struct {
    dir string          //当前日志文件目录
    oldDir string       //
    name string     //文件名 日志文件= name + 时间格式化 + . + fileSuffix
    fileSuffix string       //后缀
    interval int32      //生成新文件的间隔(秒)
    maxLogFile int32    //当前目录最多生成多少日志文件(多的移动到old)
    level LogLevel
    curPrefixLevel LogLevel

    console bool                //是否输出到console
    logFileNameList list.List   //存放日志文件列表
    mu sync.Mutex
    fd *os.File
    fn string                   //当前日志文件的文件名
    log *log.Logger
}


var stLog *STLogger = nil


func Initialize(dir, oldDir, name, suffix string, flag int, interval, maxLogFile int32) {
    if stLog != nil {
        return
    }
    log.SetFlags(flag)
    stLog = NewLogger(dir, oldDir, name, suffix, interval, maxLogFile)
}


func SetLogLevel(level LogLevel) {
    if stLog == nil {
        return
    }

    stLog.SetLevel(level)
}


func SetOutConsole(bl bool) {
    if stLog == nil {
        return
    }

    stLog.SetOutConsole(bl)
}


func Debug(fmt string, v ...interface{}) {
    if stLog == nil {
        return
    }

    stLog.Debug(fmt, v...)
}

func Warn(fmt string, v ...interface{}) {
    if stLog == nil {
        return
    }

    stLog.Warn(fmt, v...)
}

func Error(fmt string, v ...interface{}) {
    if stLog == nil {
        return
    }

    stLog.Error(fmt, v...)
}

func Info(fmt string, v ...interface{}) {
    if stLog == nil {
        return
    }

    stLog.Info(fmt, v...)
}

func Fatal(fmt string, v ...interface{}) {
    if stLog == nil {
        return
    }

    stLog.Fatal(fmt, v...)
}



func NewLogger(dir, oldDir, name, suffix string, interval,maxLogFile int32) *STLogger {
    moveAll(dir, oldDir)
    fileName := fileName(name, suffix)
    fd := createLogFd(dir, fileName)

    sysLog := log.New(fd, getLevelString(normalLevel), log.Lshortfile | log.LstdFlags)

    l := &STLogger{
        dir: dir,
        oldDir: oldDir,
        name: name,
        fileSuffix: suffix,
        interval: interval,
        maxLogFile: maxLogFile,
        level: normalLevel,
        curPrefixLevel: normalLevel,
        console: false,
        fd : fd,
        fn : fileName,
        log: sysLog,
    }

    //l.logFileNameList.PushFront(fileName)

    go l.tick()

    return l
}


func (self *STLogger)tick() {
    time.Sleep(time.Duration(self.interval) * time.Second)
    //时间到了生成新的文件
    fileName := fileName(self.name, self.fileSuffix)
    newFd := createLogFd(self.dir, fileName)

    oldFn := self.fn

    self.log.SetOutput(newFd)
    self.fd.Close()                 //关闭之前的文件
    self.fd = newFd                 //新的文件fd
    self.fn = fileName              //新的文件名

    //重命名当前文件
    hisFileName := fmt.Sprintf("%s__%s.%s", oldFn[:len(oldFn) - len(self.fileSuffix) - 1], time.Now().Format("2006-01-02_15.04.05"), self.fileSuffix)
    err := os.Rename(preDir(self.dir) + oldFn, preDir(self.dir) + hisFileName)
    if err != nil {
        log.Panicf("rename failed, %s", err.Error())
    }


    if self.logFileNameList.Len() == int(self.maxLogFile) {
        //文件数量达到最大了，移到备用目录
        t := self.logFileNameList.Back()
        e := self.logFileNameList.Remove(t)
        tailFileName := e.(string)

        err := os.Rename(preDir(self.dir) + tailFileName, preDir(self.oldDir) + tailFileName)
        if err != nil {
            log.Panicf("rename failed, %s", err.Error())
        }
    }
    self.logFileNameList.PushFront(hisFileName)


    go self.tick()
}


func (self *STLogger)SetLevel(level LogLevel) {
    self.mu.Lock()
    defer self.mu.Unlock()

    self.level = level
}


func (self *STLogger)SetOutConsole(bl bool) {
    self.console = bl
}

func (self *STLogger)Debug(fmt string, v ...interface{}) {
    self.printf(LogDebug, fmt, v...)
}

func (self *STLogger)Warn(fmt string, v ...interface{}) {
    self.printf(LogWarn, fmt, v...)
}

func (self *STLogger)Error(fmt string, v ...interface{}) {
    self.printf(LogError, fmt, v...)
}

func (self *STLogger)Info(fmt string, v ...interface{}) {
    self.printf(LogInfo, fmt, v...)
}

func (self *STLogger)Fatal(fmt string, v ...interface{}) {
    self.printf(LogFatal, fmt, v...)
}


func (self *STLogger)printf(level LogLevel, format string, v ...interface{}) {
    self.mu.Lock()
    if self.level > level {
        self.mu.Unlock()
        return
    }

    if self.curPrefixLevel != level {
        self.curPrefixLevel = level
        self.log.SetPrefix(getLevelString(level))
    }

    self.mu.Unlock()

    self.log.Output(4, fmt.Sprintf(format, v...))
    if self.console {
        log.Output(4, fmt.Sprintf(format, v...))
    }

    if level == LogFatal {
        os.Exit(1)
    }
}


func fileName(fileName, fileSuffix string ) string {
    timeStr := time.Now().Format("2006-01-02_15.04.05")
    return  fmt.Sprintf("%s%s.%s", fileName, timeStr, fileSuffix)
}


func createLogFd(dir, fileName string) *os.File {
    filePath := fmt.Sprintf("%s%s", preDir(dir), fileName)
    f, err := os.Create(filePath)
    if err != nil {
        log.Panicf("createLogFd failed, %s", err.Error())
        return nil
    }
    return f
}

func preDir(dir string) string {
    if dir == "" {
        return dir
    }

    if os.IsPathSeparator(dir[len(dir) - 1]) {
        return dir
    }
    //末尾添加分隔符
    return string(append([]byte(dir), os.PathSeparator))
}


func moveAll(sDir, dDir string) {
    allFile, err := ioutil.ReadDir(sDir)
    if err != nil {
        log.Panicf("moveAll failed, %s", err.Error())
    }


    for _, fi := range allFile {
        if !fi.IsDir() {
            name := fi.Name()
            err = os.Rename(preDir(sDir) + name, preDir(dDir) + name)
            if err != nil {
                log.Panicf("rename failed, %s", err.Error())
            }
        }
    }
}

func getLevelString(level LogLevel) string {
    switch level {
    case LogDebug:
        return "[debug] "
    case LogWarn:
        return "[warn] "
    case LogError:
        return "[error] "
    case LogInfo:
        return "[info] "
    case LogFatal:
        return "[fatal] "
    default:
        return "[*] "
    }
}