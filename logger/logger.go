package logger

import (
	"fmt"
	"kubefasthdfs/config"
	"os"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(logConfig config.LogConfigs) error {
	initFields := getInitFields(&logConfig)
	encoder := getLogEncoder()
	writer := getLogWriter(&logConfig)
	level := getLogLevel(&logConfig)
	core := zapcore.NewCore(encoder, writer, level)
	caller := zap.AddCaller()
	// 开启文件及行号
	development := zap.Development()
	// 构造日志
	Logger = zap.New(core, caller, development, zap.Fields(initFields...))
	return nil
}

func getLogWriter(logConfig *config.LogConfigs) zapcore.WriteSyncer {
	fileName := fmt.Sprintf("./logs/%s-%s.log", logConfig.ServiceName, logConfig.InstanceName)
	lumberJackLogger := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    logConfig.MaxSize,
		MaxBackups: logConfig.MaxBackups,
		MaxAge:     logConfig.MaxBackups,
		Compress:   logConfig.Compress,
	}
	if logConfig.StdOut {
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(lumberJackLogger),
			zapcore.AddSync(os.Stdout))
	} else {
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(lumberJackLogger))
	}
}

func getLogEncoder() zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "lineNum",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,  // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder, //
		EncodeCaller:   zapcore.FullCallerEncoder,      // 全路径编码器
		EncodeName:     zapcore.FullNameEncoder,
	}
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getLogLevel(logConfig *config.LogConfigs) zap.AtomicLevel {
	var level zapcore.Level
	switch logConfig.Level {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(level)
	return atomicLevel
}

func getInitFields(logConfig *config.LogConfigs) (fields []zap.Field) {
	fields = append(fields, zap.String("service", logConfig.ServiceName))
	if len(logConfig.InstanceName) == 0 {
		logConfig.InstanceName, _ = os.Hostname()
	}
	fields = append(fields, zap.String("instance", logConfig.InstanceName))
	return fields
}
