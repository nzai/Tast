package trading

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nzai/Tast/config"
	"github.com/nzai/Tast/stock"
)

const (
	dataFileName = "TradingSystem.txt"
)

//	海龟交易系统参数
type TurtleTradingSystemParameter struct {
	Holding int
	N       int
	Enter   int
	Exit    int
	Stop    int
}

//	海龟交易系统
type TurtleTradingSystem struct {
	Codes                []string
	StartAmount          float64
	Commission           float64
	StartDate            string
	EndDate              string
	Start                TurtleTradingSystemParameter
	End                  TurtleTradingSystemParameter
	Current              TurtleTradingSystemParameter
	CurrentProfit        float64
	CurrentProfitPercent float64
	Best                 TurtleTradingSystemParameter
	BestProfit           float64
	BestProfitPercent    float64
	CalculatingAmount    int64
	CalculatedAmount     int64
	CalculatedSeconds    int64
	RemainTips           string
}

func Default() *TurtleTradingSystem {
	stocks, err := stock.GetAll()
	if err != nil {
		log.Fatal("获取股票列表时发生错误:", err)
		return nil
	}

	codes := make([]string, 0)
	for _, s := range stocks {
		codes = append(codes, s.Code)
	}

	system := &TurtleTradingSystem{
		Codes:       codes,
		StartAmount: 100000,
		Commission:  7,
		StartDate:   "20060101",
		EndDate:     "20141231",
		Start: TurtleTradingSystemParameter{
			Holding: 2,
			N:       2,
			Enter:   2,
			Exit:    2,
			Stop:    2},
		End: TurtleTradingSystemParameter{
			Holding: 20,
			N:       50,
			Enter:   50,
			Exit:    50,
			Stop:    50},
		Current: TurtleTradingSystemParameter{
			Holding: 2,
			N:       2,
			Enter:   2,
			Exit:    2,
			Stop:    2},
		CurrentProfit:        0,
		CurrentProfitPercent: 0,
		Best: TurtleTradingSystemParameter{
			Holding: 2,
			N:       2,
			Enter:   2,
			Exit:    2,
			Stop:    2},
		BestProfit:        0,
		BestProfitPercent: 0,
		CalculatedSeconds: 0,
		RemainTips:        "计算尚未开始",
	}

	system.CalculatingAmount = int64(len(system.Codes) *
		(system.End.Holding - system.Start.Holding + 1) *
		(system.End.N - system.Start.N + 1) *
		(system.End.Enter - system.Start.Enter + 1) *
		(system.End.Exit - system.Start.Exit + 1) *
		(system.End.Stop - system.Start.Stop + 1))

	return system
}

var currentTurtleTradingSystem *TurtleTradingSystem = Default()

func saveSystem() error {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(dataDir, dataFileName)
	//	打开文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0x777)
	if err != nil {
		return err
	}
	defer file.Close()

	file.WriteString(fmt.Sprintf("Codes = %d %v\n", len(currentTurtleTradingSystem.Codes), currentTurtleTradingSystem.Codes))
	file.WriteString(fmt.Sprintf("StartAmount = %f\n", currentTurtleTradingSystem.StartAmount))
	file.WriteString(fmt.Sprintf("Commission = %f\n", currentTurtleTradingSystem.Commission))
	file.WriteString(fmt.Sprintf("StartDate = %s\n", currentTurtleTradingSystem.StartDate))
	file.WriteString(fmt.Sprintf("EndDate = %s\n", currentTurtleTradingSystem.EndDate))
	file.WriteString(fmt.Sprintf("Start\t[Holding = %d N = %d Enter = %d Exit = %d Stop = %d]\n",
		currentTurtleTradingSystem.Start.Holding,
		currentTurtleTradingSystem.Start.N,
		currentTurtleTradingSystem.Start.Enter,
		currentTurtleTradingSystem.Start.Exit,
		currentTurtleTradingSystem.Start.Stop))
	file.WriteString(fmt.Sprintf("End\t[Holding = %d N = %d Enter = %d Exit = %d Stop = %d]\n",
		currentTurtleTradingSystem.End.Holding,
		currentTurtleTradingSystem.End.N,
		currentTurtleTradingSystem.End.Enter,
		currentTurtleTradingSystem.End.Exit,
		currentTurtleTradingSystem.End.Stop))
	file.WriteString(fmt.Sprintf("Current\t[Holding = %d N = %d Enter = %d Exit = %d Stop = %d]\n",
		currentTurtleTradingSystem.Current.Holding,
		currentTurtleTradingSystem.Current.N,
		currentTurtleTradingSystem.Current.Enter,
		currentTurtleTradingSystem.Current.Exit,
		currentTurtleTradingSystem.Current.Stop))
	file.WriteString(fmt.Sprintf("CurrentProfit = %.3f\n", currentTurtleTradingSystem.CurrentProfit))
	file.WriteString(fmt.Sprintf("CurrentProfit = %.3f%%\n", currentTurtleTradingSystem.CurrentProfitPercent*100))
	file.WriteString(fmt.Sprintf("Best\t[Holding = %d N = %d Enter = %d Exit = %d Stop = %d]\n",
		currentTurtleTradingSystem.Best.Holding,
		currentTurtleTradingSystem.Best.N,
		currentTurtleTradingSystem.Best.Enter,
		currentTurtleTradingSystem.Best.Exit,
		currentTurtleTradingSystem.Best.Stop))
	file.WriteString(fmt.Sprintf("BestProfit = %.3f\n", currentTurtleTradingSystem.BestProfit))
	file.WriteString(fmt.Sprintf("BestProfit = %.3f%%\n", currentTurtleTradingSystem.BestProfitPercent*100))
	file.WriteString(fmt.Sprintf("CalculatingAmount = %d\n", currentTurtleTradingSystem.CalculatingAmount))
	file.WriteString(fmt.Sprintf("CalculatedAmount = %d\n", currentTurtleTradingSystem.CalculatedAmount))
	file.WriteString(fmt.Sprintf("CalculatedSeconds = %d\n", currentTurtleTradingSystem.CalculatedSeconds))
	file.WriteString(fmt.Sprintf("RemainTips = %s\n", currentTurtleTradingSystem.RemainTips))

	return nil
}

func TestAll() error {
	log.Print("开始测试海龟交易系统")

	//	保存系统
	err := saveSystem()
	if err != nil {
		return err
	}

	log.Print("海龟交易系统测试结束")

	return nil
}

func TestStock(code string) error {
	return nil
}
