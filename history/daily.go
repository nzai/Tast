package history

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nzai/Tast/config"
	"github.com/nzai/Tast/stock"
)

const (
	historyDirName        = "History"
	dailyDataFileName     = "Daily.txt"
	updateGoroutinesCount = 8
)

//	股票历史
type DailyHistory struct {
	Code     string
	Date     string
	PrevDate string
	Open     float64
	Close    float64
	High     float64
	Low      float64
	Volume   int64
}

type StockDailyHistories []DailyHistory

func (slice StockDailyHistories) Len() int {
	return len(slice)
}

func (slice StockDailyHistories) Less(i, j int) bool {
	return slice[i].Date < slice[j].Date
}

func (slice StockDailyHistories) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

//	更新股票历史
func UpdateAll() error {

	log.Print("开始更新股票历史")

	//	数据保存目录
	dataDir, err := config.GetDataDir()
	if err != nil {
		return err
	}

	//	获取所有的股票
	stocks, err := stock.GetAll()
	if err != nil {
		return err
	}

	chanSend := make(chan int, updateGoroutinesCount)
	chanReceive := make(chan int)

	//	并发获取股票历史
	go func() {
		for _, stock := range stocks {
			go func(code string) {
				//	更新每只股票的历史
				err = updateStock(code, dataDir)
				if err != nil {
					log.Fatal(err)
				}
				<-chanSend
				chanReceive <- 1
			}(stock.Code)

			chanSend <- 1
		}
	}()

	//	阻塞，直到所有股票更新完历史
	for _, _ = range stocks {
		<-chanReceive
	}

	log.Print("股票历史更新成功")

	return err
}

//	更新股票历史
func updateStock(code string, dataDir string) error {
	//log.Print(code)
	dir := filepath.Join(dataDir, code)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.Mkdir(dir, 0x777)
		if err != nil {
			return err
		}
	}

	return updateStockDaily(code, dir)
}

//	更新股票每日历史
func updateStockDaily(code string, codeDataDir string) error {

	//	每日历史文件
	filePath := filepath.Join(codeDataDir, dailyDataFileName)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		//	如果文件不存在就从纳斯达克更新股票复权每日历史
		_, err := getFromNasdaq(code, filePath)
		if err != nil {
			return err
		}
	}

	return nil
}

//	从纳斯达克更新股票复权每日历史
func getFromNasdaq(code string, filePath string) ([]DailyHistory, error) {

	//	获取记录股票历史股价的纳斯达克页面
	html, err := downloadHtmlFromNasdaq(code)
	if err != nil {
		return nil, err
	}

	//	从html中抓取股票历史股价
	histories, err := parseHtml(code, html)
	if err != nil {
		return nil, err
	}

	//	保存
	err = saveToFile(code, histories, filePath)
	if err != nil {
		return nil, err
	}

	return histories, nil
}

//	获取记录股票历史股价的纳斯达克页面
func downloadHtmlFromNasdaq(code string) (string, error) {
	queryPattern := `http://www.nasdaq.com/symbol/%s/historical`

	//	查询最近10年的除权股价及交易量
	url := fmt.Sprintf(queryPattern, strings.ToLower(code))
	payload := []byte(fmt.Sprintf("10y|false|%s", code))
	//	log.Printf("url:%s   payload:%s", url, payload)

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	//	client.Timeout = time.Second * 60
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	buffer, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(buffer), nil
}

//	从html中抓取股票历史股价
func parseHtml(code string, html string) ([]DailyHistory, error) {

	matchPattern := `<tr>\s+<td>\s+([\d\/]+)\s+</td>\s+<td>\s+([\d\.]+)\s+</td>\s+<td>\s+([\d\.]+)\s+</td>\s+<td>\s+([\d\.]+)\s+</td>\s+<td>\s+([\d\.]+)\s+</td>\s+<td>\s+([\d\.,]+)\s+</td>\s+</tr>`

	regex := regexp.MustCompile(matchPattern)
	matches := regex.FindAllStringSubmatch(html, -1)
	readLayout := "01/02/2006"
	writeLayout := "20060102"

	//	log.Print(len(matches))
	histories := make([]DailyHistory, 0)
	for _, match := range matches {
		if len(match) != 7 {
			return nil, errors.New("纳斯达克股票历史格式不正确" + fmt.Sprint(match))
		}

		date, err := time.Parse(readLayout, match[1])
		if err != nil {
			return nil, err
		}

		//		log.Print(match)
		open, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			return nil, err
		}

		high, err := strconv.ParseFloat(match[3], 64)
		if err != nil {
			return nil, err
		}

		low, err := strconv.ParseFloat(match[4], 64)
		if err != nil {
			return nil, err
		}

		_close, err := strconv.ParseFloat(match[5], 64)
		if err != nil {
			return nil, err
		}

		volume, err := strconv.ParseInt(strings.Replace(match[6], ",", "", -1), 10, 64)
		if err != nil {
			return nil, err
		}

		histories = append(histories, DailyHistory{
			Code:   code,
			Date:   date.Format(writeLayout),
			Open:   open,
			Close:  _close,
			High:   high,
			Low:    low,
			Volume: volume,
		})
	}

	//	下载的数据是日期倒序排序的，需要重新排序一下
	for index, _ := range histories {
		if index == len(histories)-1 {
			histories[index].PrevDate = ""
		} else {
			histories[index].PrevDate = histories[index+1].Date
		}
	}

	//	将股票历史按照日期正序排序
	sort.Sort(StockDailyHistories(histories))

	return histories, nil
}

//	保存股票历史
func saveToFile(code string, histories []DailyHistory, filePath string) error {
	//	打开文件
	file, err := os.OpenFile(filePath, os.O_CREATE, 0x777)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, history := range histories {

		line := fmt.Sprintf("%s\t%.6f\t%.6f\t%.6f\t%.6f\t%d\t%s\n",
			history.Date,
			history.Open,
			history.Close,
			history.High,
			history.Low,
			history.Volume,
			history.PrevDate)

		//	将股价写入文件
		_, err = file.WriteString(line)
		if err != nil {
			return err
		}
	}

	return nil
}

//	获取文件每日历史
func GetStockDailyHistory(code, dataDir string) ([]DailyHistory, error) {

	codeDailyFileName := filepath.Join(dataDir, code, dailyDataFileName)

	_, err := os.Stat(codeDailyFileName)
	if os.IsNotExist(err) {
		//	如果文件不存在就从纳斯达克获取股票复权每日历史
		return getFromNasdaq(code, codeDailyFileName)
	}

	return loadFromFile(code, codeDailyFileName)
}

//	从文件读取股票每日历史
func loadFromFile(code, filePath string) ([]DailyHistory, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	histories := make([]DailyHistory, 0)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 7 {
			return nil, errors.New("股票列表文件格式不正确")
		}

		open, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, err
		}

		_close, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return nil, err
		}

		high, err := strconv.ParseFloat(parts[3], 64)
		if err != nil {
			return nil, err
		}

		low, err := strconv.ParseFloat(parts[4], 64)
		if err != nil {
			return nil, err
		}

		volume, err := strconv.ParseInt(parts[5], 10, 64)
		if err != nil {
			return nil, err
		}

		histories = append(histories, DailyHistory{
			Code:     code,
			Date:     parts[0],
			PrevDate: parts[6],
			Open:     open,
			Close:    _close,
			High:     high,
			Low:      low,
			Volume:   volume,
		})
	}

	return histories, nil
}
