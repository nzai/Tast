package peroidexterma

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

type PeroidExtermaIndex struct {
	Code   string
	Peroid int
	Date   string
	Min    float64 //	最小值
	Max    float64 //	最大值
}

const (
	peroidMin    = 2
	peroidMax    = 50
	dataFileName = "PeroidExterma.txt"
)

//	更新区间极值指数
func UpdateAll() error {

	log.Println("开始更新区间极值指标")

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

	log.Println("区间极值指标更新完毕")

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

	allIndex := make(map[int][]PeroidExtermaIndex)
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

//	获取股票历史的最大最小值
func peroidExterma(histories []history.DailyHistory) (float64, float64) {
	min, max := math.MaxFloat64, -math.MaxFloat64
	for _, history := range histories {
		if history.Low < min {
			min = history.Low
		}

		if history.High > max {
			max = history.High
		}
	}

	return min, max
}

//	根据股价历史计算指标
func calculate(histories []history.DailyHistory, peroid int) ([]PeroidExtermaIndex, error) {

	var min, max float64
	list := make([]PeroidExtermaIndex, 0)
	queue := make([]history.DailyHistory, 0)
	for index, history := range histories {

		if index >= peroid {
			queue = append(queue[1:], history)
		} else {
			queue = append(queue, history)
		}

		if index == 0 {
			min, max = history.Low, history.High
		} else {
			min, max = peroidExterma(queue)
		}

		list = append(list, PeroidExtermaIndex{
			Code:   history.Code,
			Peroid: peroid,
			Date:   history.Date,
			Min:    min,
			Max:    max,
		})
	}

	return list, nil
}

//	将指标保存到文件
func save(code string, allIndex map[int][]PeroidExtermaIndex, filePath string) error {
	//	打开文件
	file, err := os.OpenFile(filePath, os.O_CREATE, 0x777)
	if err != nil {
		return err
	}
	defer file.Close()

	for peroid := peroidMin; peroid <= peroidMax; peroid++ {

		indexes, found := allIndex[peroid]
		if !found {
			return errors.New(fmt.Sprintf("保存区间极值指标时发现缺失code=%s peroid=%d的指标", code, peroid))
		}

		for _, index := range indexes {
			line := fmt.Sprintf("%d\t%s\t%.6f\t%.6f\n",
				index.Peroid,
				index.Date,
				index.Max,
				index.Min)

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
func load(code, filePath string) (map[int][]PeroidExtermaIndex, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	allIndex := make(map[int][]PeroidExtermaIndex)
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

		max, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return nil, err
		}

		min, err := strconv.ParseFloat(parts[3], 64)
		if err != nil {
			return nil, err
		}

		indexes, found := allIndex[peroid]
		if !found {
			allIndex[peroid] = make([]PeroidExtermaIndex, 0)
			indexes = allIndex[peroid]
		}

		indexes = append(indexes, PeroidExtermaIndex{
			Code:   code,
			Peroid: peroid,
			Date:   parts[1],
			Min:    min,
			Max:    max,
		})
	}

	return allIndex, nil
}
