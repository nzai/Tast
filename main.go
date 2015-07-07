package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/nzai/Tast/config"
	"github.com/nzai/Tast/history"
	"github.com/nzai/Tast/peroidexterma"
	"github.com/nzai/Tast/stock"
	"github.com/nzai/Tast/trading"
	"github.com/nzai/Tast/turtle"
)

const (
	configFileName           = "config.ini"
	configLogSection         = "path"
	configLogKey             = "logpath"
	configLogDefaultFileName = "main.log"
)

func main() {

	//	当前目录
	root := filepath.Dir(os.Args[0])
	filename := filepath.Join(root, configFileName)

	//	使用所有cpu
	//	runtime.GOMAXPROCS(runtime.NumCPU() - 1)

	//	读取配置文件
	err := config.SetConfigFile(filename)
	if err != nil {
		log.Fatal(err)
		return
	}

	//	日志文件路径
	logPath := config.GetString(configLogSection, configLogKey, configLogDefaultFileName)
	logDir := filepath.Dir(logPath)
	_, err = os.Stat(logDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(logDir, 0x777)
		if err != nil {
			log.Fatal(err)
			return
		}
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0x777)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()

	//	设置日志输出文件
	log.SetOutput(file)

	//	更新股票信息
	err = stock.UpdateAll()
	if err != nil {
		log.Fatalf("更新股票列表发生错误:%v", err)
		return
	}

	//	更新所有股票的历史
	err = history.UpdateAll()
	if err != nil {
		log.Fatalf("更新股票历史发生错误:%v", err)
		return
	}

	//	更新所有股票的海龟指标
	err = turtle.UpdateAll()
	if err != nil {
		log.Fatalf("更新海龟指标发生错误:%v", err)
		return
	}

	//	更新所有股票的区间极值指标
	err = peroidexterma.UpdateAll()
	if err != nil {
		log.Fatalf("更新区间极值指标发生错误:%v", err)
		return
	}

	//	测试海龟交易系统
	err = trading.TestAll()
	if err != nil {
		log.Fatalf("测试海龟交易系统发生错误:%v", err)
		return
	}
}
