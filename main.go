package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
)

type Stat struct {
	Query struct {
		Ids           []int         `json:"ids"`
		Dimensions    []interface{} `json:"dimensions"`
		Metrics       []string      `json:"metrics"`
		Sort          []string      `json:"sort"`
		Date1         string        `json:"date1"`
		Date2         string        `json:"date2"`
		Limit         int           `json:"limit"`
		Offset        int           `json:"offset"`
		Group         string        `json:"group"`
		AutoGroupSize string        `json:"auto_group_size"`
		Quantile      string        `json:"quantile"`
		Attribution   string        `json:"attribution"`
		Currency      string        `json:"currency"`
	} `json:"query"`
	Data []struct {
		Dimensions []interface{} `json:"dimensions"`
		Metrics    []float64     `json:"metrics"`
	} `json:"data"`
	TotalRows        int       `json:"total_rows"`
	TotalRowsRounded bool      `json:"total_rows_rounded"`
	Sampled          bool      `json:"sampled"`
	SampleShare      float64   `json:"sample_share"`
	SampleSize       float64   `json:"sample_size"`
	SampleSpace      float64   `json:"sample_space"`
	DataLag          float64   `json:"data_lag"`
	Totals           []float64 `json:"totals"`
	Min              []float64 `json:"min"`
	Max              []float64 `json:"max"`
}

type Stats struct {
	Statistics []Stat
	Weights    []map[string]float64
}

var (
	statsMap map[string]Stat
)

func main() {
	fmt.Println("hello")
	runtime.GOMAXPROCS(1)
	statsMap = make(map[string]Stat)
	goal := "33626871"
	ids := "42351524"
	/*
		tm := time.Now()
		year, month, day := tm.Date()
		date := fmt.Sprintf("%d.%d.%d", day, month, year)
		key := date + goal
	*/
	key := goal
	statsMap[key] = getStat(ids, goal)
	fmt.Println(statsMap[key])
	http.HandleFunc("/", handler)
	http.ListenAndServe(":9098", nil)
}

//https://api-metrika.yandex.ru/stat/v1/data?ids=42351524&metrics=ym:s:goal33626877userConversionRate&accuracy=full&metrics=ym:s:goal33626868userConversionRate&metrics=ym:s:goal33626874userConversionRate&metrics=ym:s:goal33626871userConversionRate
func getStat(ids string, goal string) (result Stat) {

	metric := fmt.Sprintf("https://api-metrika.yandex.ru/stat/v1/data?ids=%s&accuracy=full&metrics=ym:s:goal%suserConversionRate", ids, goal)

	client := &http.Client{}
	req, err := http.NewRequest("GET", metric, nil)
	req.Header.Add("Authorization", "OAuth AQAAAAABPHfIAASMOdOyz28wukRnj2GsM3UjD48")
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err == nil {
		//fmt.Println(string(body))
		err := json.Unmarshal(body, &result)
		if err != nil {
			fmt.Println(err)
		}
	}
	return result
}

func handler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	//http://localhost:9098/42351524/33626877/33626868/33626874
	urlPart := strings.Split(r.URL.Path, "/")
	var id string
	var stats Stats
	for k, goal := range urlPart {
		if k == 0 {
			continue
		}
		if k == 1 {
			id = goal
			if id == "reset" {
				for k := range statsMap {
					delete(statsMap, k)
				}
				break
			}
			continue
		}
		stat, ok := statsMap[goal]

		if ok {
			stats.Statistics = append(stats.Statistics, stat)
		} else {
			stat := getStat(id, goal)
			statsMap[goal] = stat
			fmt.Println("stat", stat)
			stats.Statistics = append(stats.Statistics, stat)
		}
	}
	if stats.Statistics != nil {
		var metrica string
		var min, sum float64
		min = 1

		for _, s := range stats.Statistics {
			if s.Query.Metrics != nil && len(s.Query.Metrics) > 0 && s.Totals != nil && len(s.Totals) > 0 {
				metrica = s.Query.Metrics[0]
				fmt.Println(metrica)
				metrica = strings.Replace(metrica, "ym:s:goal", "", -1)
				metrica = strings.Replace(metrica, "userConversionRate", "", -1)
				mapp := make(map[string]float64)

				mapp[metrica] = s.Totals[0]
				sum += s.Totals[0]
				if s.Totals[0] > 0 && s.Totals[0] < min {
					min = s.Totals[0]
				}
				stats.Weights = append(stats.Weights, mapp)
			}

		}
		fmt.Println(min, sum)
		if sum == 0 {
			sum = float64(len(stats.Weights))
		}
		if min == 0 {
			min = float64(1)
		}

		var newW []map[string]float64
		for _, v := range stats.Weights {
			for k, vv := range v {
				if vv < min {
					vv = min
				}
				mapp := make(map[string]float64)
				mapp[k] = vv / sum
				newW = append(newW, mapp)
				fmt.Println(k, vv/sum)
			}
		}
		stats.Weights = newW
	}

	b, err := json.Marshal(stats)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}

}
