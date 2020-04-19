/* Taiwan air quality App for MacOS
 * LastUpdate: 200416
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
const refreshTime = time.Hour
const url = "http://opendata2.epa.gov.tw/AQI.json"
const website = "https://airtw.epa.gov.tw/CHT/EnvMonitoring/Central/CentralMonitoring.aspx"

var (
	textAQI        string
	updateTime     string
	assignSiteName string
	siteNameList   = []string{}
	dataAQI        = make(map[string]gjson.Result)
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

		siteNameList = []string{}
		gjson.Parse(string(respBody)).ForEach(func(k, v gjson.Result) bool {
			// collect all site name
			siteNameList = append(siteNameList, v.Map()["SiteName"].String())
			dataAQI[v.Map()["SiteName"].String()] = v
			return true
		})
		return
	}
	log.Println("Connection fail")
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

	getAreaList := func() []menuet.MenuItem {
		areaItems := []menuet.MenuItem{}
		for _, k := range siteNameList {
			siteName := k
			areaItems = append(areaItems, menuet.MenuItem{
				Text: k,
				Clicked: func() {
					menuet.Defaults().SetString("location", siteName)
					refreshMenu(dataAQI[siteName].Map())
				},
				State: k == menuet.Defaults().String("location"),
			})
		}
		return areaItems
	}

	items = append(items, menuet.MenuItem{
		Text:     "Changed Area(換觀測站)",
		Children: getAreaList,
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
			"%s:%s, %s (AQI)",
			airInf["SiteName"].String(), airInf["Status"].String(),
			aqi)
	} else {
		textAQI = "連線失敗..."
	}
	// 2nd menu bar: Updated time
	updateTime = fmt.Sprintf(
		"Updated:%s", time.Now().Format("01-02 03:04:05"))
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
	siteName := menuet.Defaults().String("location")
	if siteName == "" {
		menuet.Defaults().SetString("location", "基隆")
	}

	go timerAQI() // start timer

	// Configure the application
	menuet.App().Label = "com.github.grtfou.aiq-taiwan"

	menuet.App().Children = menuItems
	menuet.App().RunApplication()
}
