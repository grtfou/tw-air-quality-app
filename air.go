/* Taiwan air quality App for MacOS
 * LastUpdate: 210428
 */

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/caseymrm/menuet"
	"github.com/tidwall/gjson"
)

// Query data frequencies
const (
	refreshTime = time.Hour
	url         = "http://opendata2.epa.gov.tw/AQI.json"
	website     = "https://airtw.epa.gov.tw/CHT/EnvMonitoring/Central/CentralMonitoring.aspx"
)

var (
	textAQI    string
	updateTime string
	// regions      = make(map[string]interface{})  // get from region.json
	siteNameList   = make(map[int][]string)        // region groups
	dataAQI        = make(map[string]gjson.Result) // all AQI data
	defaultStation = "汐止"
	regions        = map[string]int{
		"基隆市": 0,
		"新北市": 0,
		"臺北市": 0,
		"桃園市": 0,
		"新竹市": 0,
		"新竹縣": 0,
		"宜蘭縣": 0,
		"苗栗縣": 1,
		"臺中市": 1,
		"彰化縣": 1,
		"南投縣": 1,
		"雲林縣": 1,
		"嘉義市": 2,
		"嘉義縣": 2,
		"臺南市": 2,
		"高雄市": 2,
		"屏東縣": 2,
		"澎湖縣": 2,
		"花蓮縣": 3,
		"臺東縣": 3,
		"金門縣": 4,
		"連江縣": 4}
)

func getAQI() {
	resp, err := http.Get(url)
	if e, ok := err.(net.Error); ok && e.Timeout() {
		log.Println("Connection timeout")
		return
	} else if err != nil {
		log.Println("Connection fail")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return
		}

		// Group by regions
		// https://en.wikipedia.org/wiki/Regions_of_Taiwan
		siteList0 := []string{}
		siteList1 := []string{}
		siteList2 := []string{}
		siteList3 := []string{}
		siteList4 := []string{}
		gjson.Parse(string(respBody)).ForEach(func(k, v gjson.Result) bool {
			// collect all area name and AQI value
			// fmt.Println(v.Map()["County"].String())
			regionGroupLabel := int(regions[v.Map()["County"].String()])
			siteName := v.Map()["SiteName"].String()
			if regionGroupLabel == 0 {
				siteList0 = append(siteList0, siteName)
			} else if regionGroupLabel == 1 {
				siteList1 = append(siteList1, siteName)
			} else if regionGroupLabel == 2 {
				siteList2 = append(siteList2, siteName)
			} else if regionGroupLabel == 3 {
				siteList3 = append(siteList3, siteName)
			} else if regionGroupLabel == 4 {
				siteList4 = append(siteList4, siteName)
			} else {
				log.Println("Not include:", siteName)
			}
			dataAQI[siteName] = v // AQI
			return true
		})
		siteNameList[0] = siteList0
		siteNameList[1] = siteList1
		siteNameList[2] = siteList2
		siteNameList[3] = siteList3
		siteNameList[4] = siteList4
		// -----
		return
	}
	log.Println("Connection fail")
}

func getStationData(region int) []menuet.MenuItem {
	areaItems := []menuet.MenuItem{}

	for _, k := range siteNameList[region] {
		siteName := k
		areaItems = append(areaItems, menuet.MenuItem{
			Text: siteName,
			Clicked: func() {
				menuet.Defaults().SetString("location", siteName)
				refreshMenu(dataAQI[siteName].Map())
			},
			State: k == menuet.Defaults().String("location"),
		})
	}
	return areaItems
}

func menuItems() []menuet.MenuItem {
	items := []menuet.MenuItem{}
	items = append(items, menuet.MenuItem{
		Text:     textAQI,
		FontSize: 15,
		Clicked: func() {
			exec.Command("open", website).Start()
		},
	})

	items = append(items, menuet.MenuItem{
		Text:     updateTime,
		FontSize: 14,
	})

	items = append(items, menuet.MenuItem{
		Type: menuet.Separator,
	})

	zoneList := func() []menuet.MenuItem {
		zoneItems := []menuet.MenuItem{}
		zoneItems = append(zoneItems, menuet.MenuItem{
			Text:     "Northern (北部)",
			Children: func() []menuet.MenuItem { return getStationData(0) },
		})
		zoneItems = append(zoneItems, menuet.MenuItem{
			Text:     "Central (中部)",
			Children: func() []menuet.MenuItem { return getStationData(1) },
		})
		zoneItems = append(zoneItems, menuet.MenuItem{
			Text:     "Southern (南部)",
			Children: func() []menuet.MenuItem { return getStationData(2) },
		})
		zoneItems = append(zoneItems, menuet.MenuItem{
			Text:     "Eastern (東部)",
			Children: func() []menuet.MenuItem { return getStationData(3) },
		})
		zoneItems = append(zoneItems, menuet.MenuItem{
			Text:     "Outer islands (離島)",
			Children: func() []menuet.MenuItem { return getStationData(4) },
		})
		return zoneItems
	}

	items = append(items, menuet.MenuItem{
		Text:     "Choose station (觀測站)",
		Children: zoneList,
	})

	return items
}

func refreshMenu(airInf map[string]gjson.Result) {
	var aqi string
	if airInf["AQI"].String() != "" {
		if airInf["AQI"].Int() >= 100 {
			aqi = "😷" + airInf["AQI"].String()
		} else {
			aqi = "🏖️" + airInf["AQI"].String()
		}
	} else {
		aqi = "-"
	}

	menuet.App().SetMenuState(&menuet.MenuState{
		Title: aqi,
	})

	if len(airInf) > 0 {
		// 1st menu bar: AQI information
		textAQI = fmt.Sprintf(
			"%s: %s, %s (AQI)",
			airInf["SiteName"].String(), airInf["Status"].String(),
			aqi)
	} else {
		textAQI = "連線失敗..."
	}
	// 2nd menu bar: Updated time
	updateTime = fmt.Sprintf(
		"Updated: %s", time.Now().Format("01-02 15:04:05"))
}

func timerAQI() {
	getAQI() // first time
	for {
		siteName := menuet.Defaults().String("location")
		refreshMenu(dataAQI[siteName].Map())
		time.Sleep(refreshTime)
		getAQI()
	}
}

func main() {
	lastSetSiteName := menuet.Defaults().String("location")
	if lastSetSiteName == "" {
		menuet.Defaults().SetString(
			"location", defaultStation)
	}

	go timerAQI() // start timer

	// Configure the application
	menuet.App().Label = "https://github.com/grtfou/tw-air-quality-app"
	menuet.App().Children = menuItems
	menuet.App().RunApplication()
}
