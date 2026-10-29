// Package udger package allow you to load in memory and lookup the user agent database to extract value from the provided user agent
package udger

import (
	"database/sql"
	"os"
	"regexp"
	"regexp/syntax"
	"strings"
	"unicode"
)

const CRAWLER_CLASS_ID = 99

// New creates a new instance of Udger from the dbPath database loaded in memory for fast lookup.
func New(dbPath string) (*Udger, error) {
	u := &Udger{
		Browsers:          make(map[int]Browser),
		OS:                make(map[int]OS),
		Devices:           make(map[int]Device),
		DevicesBrand:      make(map[int]DeviceBrand),
		DevicesMarketName: make(map[string]DeviceMarketName),
		browserTypes:      make(map[int]string),
		browserOS:         make(map[int]int),
		crawlerTypes:      make(map[int]string),
		Crawlers:          make(map[string]Crawler),
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

	lowerUA := strings.ToLower(ua)

	browserID, browserVersion := udger.findDataWithVersion(ua, udger.rexBrowsers, findWords(lowerUA, udger.wordsBrowsers), true)
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
		osID, _ := udger.findDataWithVersion(ua, udger.rexOS, findWords(lowerUA, udger.wordsOS), false)
		info.OS = udger.OS[osID]
	}

	deviceID, _ := udger.findDataWithVersion(ua, udger.rexDevices, findWords(lowerUA, udger.wordsDevices), false)

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

	// find device brand and market name
	_, s := udger.findDataWithVersion(ua, udger.rexDeviceNames, nil, true)
	if val, ok := udger.DevicesMarketName[s]; ok {
		info.DeviceMarketName = val
		info.DeviceBrand = udger.DevicesBrand[val.BrandID]
	}

	return info, nil
}

func (udger *Udger) cleanRegex(r string) string {
	// removes single-line and case-insensitive modifiers
	r = strings.TrimSuffix(r, "/si")
	r = strings.TrimPrefix(r, "/")
	return r
}

// findWords returns the IDs of the words contained in the lowercased user agent. ID 0 (no word) is always present.
// It returns nil for non-ASCII user agents: case-insensitive regexes match some non-ASCII runes as ASCII letters
// (e.g. "ſ" matches "s"), which strings.Contains would miss.
func findWords(lowerUA string, words []rexWord) map[int]bool {
	for i := 0; i < len(lowerUA); i++ {
		if lowerUA[i] > unicode.MaxASCII {
			return nil
		}
	}
	found := map[int]bool{0: true}
	for _, w := range words {
		if strings.Contains(lowerUA, w.Word) {
			found[w.ID] = true
		}
	}
	return found
}

// findDataWithVersion returns the ID of the first regex matching ua, and its first capture group if withVersion is set.
// When foundWords is not nil, regexes whose words are not all in foundWords are skipped without being evaluated.
func (udger *Udger) findDataWithVersion(ua string, data []rexData, foundWords map[int]bool, withVersion bool) (int, string) {
	index := -1
	version := ""
	for i := 0; i < len(data); i++ {
		if foundWords != nil && (!foundWords[data[i].WordID] || !foundWords[data[i].Word2ID]) {
			continue
		}
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
	rows, err := udger.db.Query("SELECT client_id, regstring, word_id, word2_id FROM udger_client_regex ORDER by sequence ASC")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d rexData
		rows.Scan(&d.ID, &d.Regex, &d.WordID, &d.Word2ID)
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

	udger.wordsBrowsers, err = udger.loadWords("udger_client_regex_words")
	if err != nil {
		return err
	}
	if err := dropUnsafeWords(udger.rexBrowsers, udger.wordsBrowsers); err != nil {
		return err
	}

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
	rows, err := udger.db.Query("SELECT deviceclass_id, regstring, word_id, word2_id FROM udger_deviceclass_regex ORDER by sequence ASC")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d rexData
		rows.Scan(&d.ID, &d.Regex, &d.WordID, &d.Word2ID)
		d.Regex = udger.cleanRegex(d.Regex)
		r, err := regexp.Compile("(?i)" + d.Regex)
		if err != nil {
			return err
		}
		d.RegexCompiled = r
		udger.rexDevices = append(udger.rexDevices, d)
	}
	rows.Close()

	udger.wordsDevices, err = udger.loadWords("udger_deviceclass_regex_words")
	if err != nil {
		return err
	}
	if err := dropUnsafeWords(udger.rexDevices, udger.wordsDevices); err != nil {
		return err
	}

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

	rows, err = udger.db.Query("SELECT id, regstring FROM udger_devicename_regex ORDER by sequence ASC")
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
		udger.rexDeviceNames = append(udger.rexDeviceNames, d)
	}
	rows.Close()

	rows, err = udger.db.Query("SELECT code, marketname, brand_id FROM udger_devicename_list")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d DeviceMarketName
		var code string
		rows.Scan(&code, &d.Name, &d.BrandID)
		d.Code = code
		udger.DevicesMarketName[code] = d
	}
	rows.Close()

	rows, err = udger.db.Query("SELECT id, brand FROM udger_devicename_brand")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d DeviceBrand
		var id int
		rows.Scan(&id, &d.Name)
		udger.DevicesBrand[id] = d
	}
	rows.Close()

	return nil
}

func (udger *Udger) initOS() error {
	rows, err := udger.db.Query("SELECT os_id, regstring, word_id, word2_id FROM udger_os_regex ORDER by sequence ASC")
	if err != nil {
		return err
	}
	for rows.Next() {
		var d rexData
		rows.Scan(&d.ID, &d.Regex, &d.WordID, &d.Word2ID)
		d.Regex = udger.cleanRegex(d.Regex)
		r, err := regexp.Compile("(?i)" + d.Regex)
		if err != nil {
			return err
		}
		d.RegexCompiled = r
		udger.rexOS = append(udger.rexOS, d)
	}
	rows.Close()

	udger.wordsOS, err = udger.loadWords("udger_os_regex_words")
	if err != nil {
		return err
	}
	if err := dropUnsafeWords(udger.rexOS, udger.wordsOS); err != nil {
		return err
	}

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

func (udger *Udger) loadWords(table string) ([]rexWord, error) {
	rows, err := udger.db.Query("SELECT id, word FROM " + table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var words []rexWord
	for rows.Next() {
		var w rexWord
		rows.Scan(&w.ID, &w.Word)
		w.Word = strings.ToLower(w.Word)
		words = append(words, w)
	}
	return words, nil
}

// dropUnsafeWords resets the words of a regex when they are not guaranteed to be in every string it matches.
// Udger words are a hint and can come from a single branch of an alternation (e.g. "fire" in "(AFTN|Fire HD10)"):
// prefiltering on them would skip regexes that do match.
func dropUnsafeWords(data []rexData, words []rexWord) error {
	wordByID := make(map[int]string, len(words))
	for _, w := range words {
		wordByID[w.ID] = w.Word
	}
	for i := range data {
		re, err := syntax.Parse("(?i)"+data[i].Regex, syntax.Perl)
		if err != nil {
			return err
		}
		literals := requiredLiterals(re)
		if !isRequiredWord(wordByID[data[i].WordID], literals) {
			data[i].WordID = 0
		}
		if !isRequiredWord(wordByID[data[i].Word2ID], literals) {
			data[i].Word2ID = 0
		}
	}
	return nil
}

// isRequiredWord reports whether word is an ASCII substring of one of the required literals.
func isRequiredWord(word string, literals []string) bool {
	if word == "" {
		return false
	}
	for i := 0; i < len(word); i++ {
		if word[i] > unicode.MaxASCII {
			return false
		}
	}
	for _, l := range literals {
		if strings.Contains(l, word) {
			return true
		}
	}
	return false
}

// requiredLiterals returns lowercased literals that every string matched by re contains.
// It is conservative: alternations, optional and repeated-zero-times parts contribute nothing.
func requiredLiterals(re *syntax.Regexp) []string {
	switch re.Op {
	case syntax.OpLiteral:
		return []string{strings.ToLower(string(re.Rune))}
	case syntax.OpCapture, syntax.OpPlus:
		return requiredLiterals(re.Sub[0])
	case syntax.OpRepeat:
		if re.Min >= 1 {
			return requiredLiterals(re.Sub[0])
		}
	case syntax.OpConcat:
		var literals []string
		run := ""
		for _, sub := range re.Sub {
			if sub.Op == syntax.OpLiteral {
				run += strings.ToLower(string(sub.Rune))
				continue
			}
			if run != "" {
				literals = append(literals, run)
				run = ""
			}
			literals = append(literals, requiredLiterals(sub)...)
		}
		if run != "" {
			literals = append(literals, run)
		}
		return literals
	}
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
