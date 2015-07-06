package stock

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nzai/Tast/config"
)

const (
	stocksFileName = "stocks.txt"
	//	从智库百科-标准普尔500指数页面下载成份股
	sp100url = "http://wiki.mbalib.com/wiki/%E6%A0%87%E5%87%86%E6%99%AE%E5%B0%94100%E6%8C%87%E6%95%B0"
)

//	股票
type Stock struct {
	Code        string
	EnglishName string
	ChineseName string
}

//	更新股票列表
func UpdateAll() error {

	log.Println("开始更新股票列表")
	//	更新股票
	_, err := GetAll()

	log.Println("股票列表更新结束")

	return err
}

//	获取股票列表
func GetAll() ([]Stock, error) {

	//	数据保存目录
	dataDir, err := config.GetDataDir()
	if err != nil {
		return nil, err
	}

	//	股票列表文件路径
	filePath := filepath.Join(dataDir, stocksFileName)
	_, err = os.Stat(filePath)
	if os.IsNotExist(err) {
		//	如果股票列表文件不存在，则从智库百科下载
		stocks, err := downloadFromMbalib()
		if err != nil {
			return nil, err
		}
		//	保存下载的股票
		return stocks, save(stocks, filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	stocks := make([]Stock, 0)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 3 {
			return nil, errors.New("股票列表文件格式不正确")
		}

		stocks = append(stocks, Stock{
			Code:        strings.ToUpper(parts[0]),
			EnglishName: parts[1],
			ChineseName: parts[2],
		})
	}

	return stocks, nil
}

//	从智库百科下载标普100成份股
func downloadFromMbalib() ([]Stock, error) {
	response, err := http.Get(sp100url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	buffer, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	regex := regexp.MustCompile(`<td>(<a[^>]*?>)?(\w+)(</a>)?</td><td><a[^>]*?>([^<]+)</a>([^<]*)?</td><td><a[^>]*?>([^<]+?)</a>`)
	matches := regex.FindAllStringSubmatch(string(buffer), -1)

	stocks := make([]Stock, 0)
	for _, match := range matches {

		if len(match) != 7 {
			return nil, errors.New("股票列表格式不正确")
		}

		stocks = append(stocks, Stock{
			Code:        match[2],
			EnglishName: match[4],
			ChineseName: match[6],
		})
	}

	return stocks, nil
}

//	保存
func save(stocks []Stock, filePath string) error {

	//	打开文件
	file, err := os.OpenFile(filePath, os.O_CREATE, 0x777)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, stock := range stocks {

		line := fmt.Sprintf("%s\t%s\t%s\n",
			stock.Code,
			stock.EnglishName,
			stock.ChineseName)

		//	将股票写入文件
		_, err = file.WriteString(line)
		if err != nil {
			return err
		}
	}
	return nil
}
