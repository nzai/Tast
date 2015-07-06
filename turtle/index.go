package turtle

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"

	"strconv"
	"strings"

	"github.com/nzai/Tast/config"
	"github.com/nzai/Tast/history"
	"github.com/nzai/Tast/stock"
)

type TurtleIndex struct {
	Code   string
	Peroid int
	Date   string
	N      float64 //	波动性均值
	TR     float64 //	真实波动性
}

const (
	peroidMin    = 2
	peroidMax    = 50
	dataFileName = "Turtle.txt"
)

//	更新海龟指数
func UpdateAll() error {

	log.Println("开始更新海龟指标")

	//	数据保存目录
	dataDir, err := config.GetDataDir()
	if err != nil {
		return err
	}

	//	获取所有股票
	stocks, err := stock.GetAll()
	if err != nil {
		return err
	}

	//log.Printf("共有股票%d只", len(stocks))

	for _, stock := range stocks {
		//	更新每只股票的指标
		err = updateStock(stock.Code, dataDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("海龟指标更新完毕")

	return err
}

func updateStock(code string, dataDir string) error {
	//	获取股票每日历史
	histories, err := history.GetStockDailyHistory(code, dataDir)
	if err != nil {
		return err
	}

	filePath := filepath.Join(dataDir, code, dataFileName)
	_, err = os.Stat(filePath)
	if !os.IsNotExist(err) {
		//	如果文件存在就跳过不重新计算
		return nil
	}
	//log.Printf("股票%s历史记录有%d天", code, len(histories))

	allIndex := make(map[int][]TurtleIndex)
	chanReceive := make(chan int)

	//	并发计算指标
	go func() {
		for peroid := peroidMin; peroid <= peroidMax; peroid++ {
			go func(p int) {
				//	更新股票在周期为peroid时的指数
				indexes, err := calculate(histories, p)
				if err != nil {
					log.Fatal(err)
				}

				allIndex[p] = indexes
				chanReceive <- 1
			}(peroid)
		}
	}()

	//	阻塞，直到所有股票更新完历史
	for peroid := peroidMin; peroid <= peroidMax; peroid++ {
		<-chanReceive
	}

	//	保存
	return save(code, allIndex, filePath)
}

//	根据股价历史计算指标
func calculate(histories []history.DailyHistory, peroid int) ([]TurtleIndex, error) {

	peroid64 := float64(peroid)
	var n, prevn, pdc, tr float64
	list := make([]TurtleIndex, 0)
	for index, history := range histories {
		if index == 0 {
			pdc = 0
		} else {
			pdc = histories[index-1].Close
		}

		tr = math.Max(history.High-history.Low, math.Max(history.High-pdc, pdc-history.Low))

		if index == 0 {
			n = tr / peroid64
		} else {
			n = ((peroid64-1)*prevn + tr) / peroid64
		}

		list = append(list, TurtleIndex{
			Code:   history.Code,
			Peroid: peroid,
			Date:   history.Date,
			N:      n,
			TR:     tr,
		})
	}

	return list, nil
}

//	将指标保存到文件
func save(code string, allIndex map[int][]TurtleIndex, filePath string) error {
	//	打开文件
	file, err := os.OpenFile(filePath, os.O_CREATE, 0x777)
	if err != nil {
		return err
	}
	defer file.Close()

	for peroid := peroidMin; peroid <= peroidMax; peroid++ {

		indexes, found := allIndex[peroid]
		if !found {
			return errors.New(fmt.Sprintf("保存海龟指标时发现缺失code=%s peroid=%d的指标", code, peroid))
		}

		for _, index := range indexes {
			line := fmt.Sprintf("%d\t%s\t%.6f\t%.6f\n",
				index.Peroid,
				index.Date,
				index.N,
				index.TR)

			//	将股价写入文件
			_, err = file.WriteString(line)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//	从文件中读入指标
func load(code, filePath string) (map[int][]TurtleIndex, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	allIndex := make(map[int][]TurtleIndex)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 4 {
			return nil, errors.New("股票列表文件格式不正确")
		}

		peroid64, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, err
		}
		peroid := int(peroid64)

		n, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return nil, err
		}

		tr, err := strconv.ParseFloat(parts[3], 64)
		if err != nil {
			return nil, err
		}

		indexes, found := allIndex[peroid]
		if !found {
			allIndex[peroid] = make([]TurtleIndex, 0)
			indexes = allIndex[peroid]
		}

		indexes = append(indexes, TurtleIndex{
			Code:   code,
			Peroid: peroid,
			Date:   parts[1],
			N:      n,
			TR:     tr,
		})
	}

	return allIndex, nil
}
