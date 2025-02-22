// Package udger package allow you to load in memory and lookup the user agent database to extract value from the provided user agent
package udger

import (
	"database/sql"
	"os"
	"regexp"
	"strings"
)

const CRAWLER_CLASS_ID = 99

// New creates a new instance of Udger from the dbPath database loaded in memory for fast lookup.
func New(dbPath string) (*Udger, error) {
	u := &Udger{
		Browsers:     make(map[int]Browser),
		OS:           make(map[int]OS),
		Devices:      make(map[int]Device),
		browserTypes: make(map[int]string),
		browserOS:    make(map[int]int),
		crawlerTypes: make(map[int]string),
		Crawlers:     make(map[string]Crawler),
	}
	var err error

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, err
	}

	u.db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer u.db.Close()

	err = u.init()
	if err != nil {
		return nil, err
	}

	return u, nil
}

// Lookup returns all the metadata possible for the given user agent string ua.
func (udger *Udger) Lookup(ua string) (*Info, error) {
	info := &Info{}

	browserID, browserVersion := udger.findDataWithVersion(ua, udger.rexBrowsers, true)
	if browser, found := udger.Browsers[browserID]; found {
		info.Browser = browser
		if info.Browser.Family != "" {
			info.Browser.Name = browser.Family + " " + browserVersion
		}
		info.Browser.Version = browserVersion
		info.Browser.Type = udger.browserTypes[browser.typ]
	} else {
		info.Browser.typ = -1
	}

	if crawler, found := udger.Crawlers[ua]; found {
		info.Crawler = crawler
		info.Crawler.Class = udger.crawlerTypes[crawler.ClassId]
		info.Browser.typ = CRAWLER_CLASS_ID
		info.Browser.Type = udger.browserTypes[CRAWLER_CLASS_ID]
	}

	if val, ok := udger.browserOS[browserID]; ok {
		info.OS = udger.OS[val]
	} else {
		osID, _ := udger.findDataWithVersion(ua, udger.rexOS, false)
		info.OS = udger.OS[osID]
	}

	deviceID, _ := udger.findDataWithVersion(ua, udger.rexDevices, false)

	if val, ok := udger.Devices[deviceID]; ok {
		info.Device = val
	} else if info.Browser.typ == -1 { // empty
		// pass
	} else if info.Browser.typ == 3 { // if browser is mobile, we can guess it's a mobile
		info.Device = Device{
			Name: "Smartphone",
			Icon: "phone.png",
		}
	} else if info.Browser.typ == 5 || info.Browser.typ == 10 || info.Browser.typ == 20 || info.Browser.typ == 50 || info.Browser.typ == CRAWLER_CLASS_ID {
		info.Device = Device{
			Name: "Other",
			Icon: "other.png",
		}
	} else {
		//nothing so personal computer
		info.Device = Device{
			Name: "Personal computer",
			Icon: "desktop.png",
		}
	}
	return info, nil
}

func (udger *Udger) cleanRegex(r string) string {
	// removes single-line and case-insensitive modifiers
	r = strings.TrimSuffix(r, "/si")
	r = strings.TrimPrefix(r, "/")
	return r
}

func (udger *Udger) findDataWithVersion(ua string, data []rexData, withVersion bool) (int, string) {
	index := -1
	version := ""
	for i := 0; i < len(data); i++ {
		r := data[i].RegexCompiled
		if !r.MatchString(ua) {
			continue
		}
		index = data[i].ID
		if withVersion {
			sub := r.FindStringSubmatch(ua)
			if len(sub) >= 2 {
				version = sub[1]
			}
		}
		break
	}
	return index, version
}

func (udger *Udger) init() error {
	if err := udger.initBrowsers(); err != nil {
		return err
	}
	if err := udger.initDevices(); err != nil {
		return err
	}
	if err := udger.initOS(); err != nil {
		return err
	}
	if err := udger.initCrawlers(); err != nil {
		return err
	}
	return nil
}

func (udger *Udger) initBrowsers() error {
	rows, err := udger.db.Query("SELECT client_id, regstring FROM udger_client_regex ORDER by sequence ASC")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d rexData
		rows.Scan(&d.ID, &d.Regex)
		d.Regex = udger.cleanRegex(d.Regex)
		// set case-insensitive flag withing current group
		r, err := regexp.Compile("(?i)" + d.Regex)
		if err != nil {
			return err
		}
		d.RegexCompiled = r
		udger.rexBrowsers = append(udger.rexBrowsers, d)
	}
	rows.Close()

	// Chrome, Safari, Firefox, etc.
	rows, err = udger.db.Query("SELECT id, class_id, name,engine,vendor,icon FROM udger_client_list")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d Browser
		var id int
		rows.Scan(&id, &d.typ, &d.Family, &d.Engine, &d.Company, &d.Icon)
		udger.Browsers[id] = d
	}
	rows.Close()

	// browser, mobile, crawler, etc.
	rows, err = udger.db.Query("SELECT id, client_classification FROM udger_client_class")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d string
		var id int
		rows.Scan(&id, &d)
		udger.browserTypes[id] = d
	}
	rows.Close()

	rows, err = udger.db.Query("SELECT client_id, os_id FROM udger_client_os_relation")
	if err != nil {
		return err
	}
	for rows.Next() {
		var browser int
		var os int
		rows.Scan(&browser, &os)
		udger.browserOS[browser] = os
	}
	rows.Close()
	return nil
}

func (udger *Udger) initDevices() error {
	rows, err := udger.db.Query("SELECT deviceclass_id, regstring FROM udger_deviceclass_regex ORDER by sequence ASC")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d rexData
		rows.Scan(&d.ID, &d.Regex)
		d.Regex = udger.cleanRegex(d.Regex)
		r, err := regexp.Compile("(?i)" + d.Regex)
		if err != nil {
			return err
		}
		d.RegexCompiled = r
		udger.rexDevices = append(udger.rexDevices, d)
	}
	rows.Close()

	rows, err = udger.db.Query("SELECT id, name, icon FROM udger_deviceclass_list")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d Device
		var id int
		rows.Scan(&id, &d.Name, &d.Icon)
		udger.Devices[id] = d
	}
	rows.Close()
	return nil
}

func (udger *Udger) initOS() error {
	rows, err := udger.db.Query("SELECT os_id, regstring FROM udger_os_regex ORDER by sequence ASC")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d rexData
		rows.Scan(&d.ID, &d.Regex)
		d.Regex = udger.cleanRegex(d.Regex)
		r, err := regexp.Compile("(?i)" + d.Regex)
		if err != nil {
			return err
		}
		d.RegexCompiled = r
		udger.rexOS = append(udger.rexOS, d)
	}
	rows.Close()

	rows, err = udger.db.Query("SELECT id, name, family, vendor, icon FROM udger_os_list")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d OS
		var id int
		rows.Scan(&id, &d.Name, &d.Family, &d.Company, &d.Icon)
		udger.OS[id] = d
	}
	rows.Close()
	return nil
}

func (udger *Udger) initCrawlers() error {
	// Uncategorised, Search engine bot, Site monitor, etc.
	rows, err := udger.db.Query("SELECT id, crawler_classification FROM udger_crawler_class")
	if err != nil {
		return err
	}
	for rows.Next() {
		var crawlerClass string
		var id int
		rows.Scan(&id, &crawlerClass)
		udger.crawlerTypes[id] = crawlerClass
	}
	rows.Close()

	rows, err = udger.db.Query("SELECT ua_string, name, family, vendor, class_id FROM udger_crawler_list")
	if err != nil {
		return err
	}
	for rows.Next() {
		var crawler Crawler
		var uaString string
		rows.Scan(&uaString, &crawler.Name, &crawler.Family, &crawler.Vendor, &crawler.ClassId)
		udger.Crawlers[uaString] = crawler
	}
	rows.Close()
	return nil
}
