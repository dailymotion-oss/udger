package udger

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"regexp/syntax"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func ExampleUdger_Lookup() {
	udger, err := New("./udgerdb_v3.dat")
	if err != nil {
		panic(err)
	}

	info, err := udger.Lookup("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2575.0 Safari/537.36")
	if err != nil {
		panic(err)
	}
	fmt.Println(info.Browser.Family, info.OS.Family)
	// output: Chrome OS X
}

func TestInvalidDbName(t *testing.T) {
	Convey("load invalid path", t, func() {
		udger, err := New("./should_not_exit.dat")
		So(err, ShouldNotBeNil)
		So(udger, ShouldBeNil)
	})
}

func TestValidDbName(t *testing.T) {
	Convey("load valid path", t, func() {
		udger, err := New("./udgerdb_v3.dat")
		So(err, ShouldBeNil)
		So(udger, ShouldNotBeNil)

		Convey("test memory database", func() {
			So(len(udger.Browsers), ShouldBeGreaterThan, 0)
			So(len(udger.Devices), ShouldBeGreaterThan, 0)
			So(len(udger.OS), ShouldBeGreaterThan, 0)

			Convey("test empty", func() {
				info, err := udger.Lookup("")
				So(err, ShouldBeNil)
				So(info, ShouldNotBeNil)

				Convey("test lookup info", func() {
					So(info.OS.Company, ShouldResemble, "")
					So(info.OS.Family, ShouldResemble, "")
					So(info.OS.Icon, ShouldResemble, "")
					So(info.OS.Name, ShouldResemble, "")

					So(info.Device.Name, ShouldResemble, "")
					So(info.Device.Icon, ShouldResemble, "")

					So(info.Browser.Company, ShouldResemble, "")
					So(info.Browser.Engine, ShouldResemble, "")
					So(info.Browser.Family, ShouldResemble, "")
					So(info.Browser.Icon, ShouldResemble, "")
					So(info.Browser.Name, ShouldResemble, "")
					So(info.Browser.Type, ShouldResemble, "")
					So(info.Browser.Version, ShouldResemble, "")
				})
			})

			Convey("test lookup MAC", func() {
				info, err := udger.Lookup("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2575.0 Safari/537.36")
				So(err, ShouldBeNil)
				So(info, ShouldNotBeNil)

				Convey("test lookup info", func() {
					So(info.OS.Company, ShouldResemble, "Apple Inc.")
					So(info.OS.Family, ShouldResemble, "OS X")
					So(info.OS.Icon, ShouldResemble, "macosx.png")
					So(info.OS.Name, ShouldResemble, "OS X 10.11 El Capitan")

					So(info.Device.Name, ShouldResemble, "Personal computer")
					So(info.Device.Icon, ShouldResemble, "desktop.png")

					So(info.Browser.Company, ShouldResemble, "Google Inc.")
					So(info.Browser.Engine, ShouldResemble, "WebKit/Blink")
					So(info.Browser.Family, ShouldResemble, "Chrome")
					So(info.Browser.Icon, ShouldResemble, "chrome.png")
					So(info.Browser.Name, ShouldResemble, "Chrome 49.0.2575.0")
					So(info.Browser.Type, ShouldResemble, "Browser")
					So(info.Browser.Version, ShouldResemble, "49.0.2575.0")
				})
			})

			Convey("test lookup Win", func() {
				info, err := udger.Lookup("Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.1; Trident/4.0; SLCC2; .NET CLR 2.0.50727; .NET CLR 3.5.30729; .NET CLR 3.0.30729; Media Center PC 6.0)")
				So(err, ShouldBeNil)
				So(info, ShouldNotBeNil)

				Convey("test lookup info", func() {
					So(info.OS.Company, ShouldResemble, "Microsoft Corporation.")
					So(info.OS.Family, ShouldResemble, "Windows")
					So(info.OS.Icon, ShouldResemble, "windows-7.png")
					So(info.OS.Name, ShouldResemble, "Windows 7")

					So(info.Device.Name, ShouldResemble, "Personal computer")
					So(info.Device.Icon, ShouldResemble, "desktop.png")

					So(info.Browser.Company, ShouldResemble, "Microsoft Corporation.")
					So(info.Browser.Engine, ShouldResemble, "Trident")
					So(info.Browser.Family, ShouldResemble, "IE")
					So(info.Browser.Icon, ShouldResemble, "msie.png")
					So(info.Browser.Name, ShouldResemble, "IE 8.0")
					So(info.Browser.Type, ShouldResemble, "Browser")
					So(info.Browser.Version, ShouldResemble, "8.0")
				})
			})

			Convey("test Nintendo DS", func() {
				info, err := udger.Lookup("Opera/9.50 (Nintendo DSi; Opera/507; U; en-US)")
				So(err, ShouldBeNil)
				So(info, ShouldNotBeNil)

				Convey("test lookup info", func() {
					So(info.OS.Company, ShouldResemble, "Nintendo of America Inc.")
					So(info.OS.Family, ShouldResemble, "Nintendo")
					So(info.OS.Icon, ShouldResemble, "nintendoDS.png")
					So(info.OS.Name, ShouldResemble, "Nintendo DS")

					So(info.Device.Name, ShouldResemble, "Game console")
					So(info.Device.Icon, ShouldResemble, "console.png")

					So(info.Browser.Company, ShouldResemble, "Opera Software ASA.")
					So(info.Browser.Engine, ShouldResemble, "Presto/Blink")
					So(info.Browser.Family, ShouldResemble, "Opera")
					So(info.Browser.Icon, ShouldResemble, "opera.png")
					So(info.Browser.Name, ShouldResemble, "Opera 9.50")
					So(info.Browser.Type, ShouldResemble, "Browser")
					So(info.Browser.Version, ShouldResemble, "9.50")
				})
			})

			Convey("test Iphone", func() {
				info, err := udger.Lookup("Mozilla/5.0 (iPhone; CPU iPhone OS 9_2_1 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Mobile/13D15")
				So(err, ShouldBeNil)
				So(info, ShouldNotBeNil)

				Convey("test lookup info", func() {
					So(info.OS.Company, ShouldResemble, "Apple Inc.")
					So(info.OS.Family, ShouldResemble, "iOS")
					So(info.OS.Icon, ShouldResemble, "iphone.png")
					So(info.OS.Name, ShouldResemble, "iOS 9")

					So(info.Device.Name, ShouldResemble, "Smartphone")
					So(info.Device.Icon, ShouldResemble, "phone.png")

					So(info.Browser.Company, ShouldResemble, "Apple Inc.")
					So(info.Browser.Engine, ShouldResemble, "WebKit")
					So(info.Browser.Family, ShouldResemble, "Safari mobile")
					So(info.Browser.Icon, ShouldResemble, "safari.png")
					So(info.Browser.Name, ShouldResemble, "Safari mobile ")
					So(info.Browser.Type, ShouldResemble, "Mobile browser")
					So(info.Browser.Version, ShouldResemble, "")
				})
			})

			Convey("test Crawler", func() {
				info, err := udger.Lookup("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
				So(err, ShouldBeNil)
				So(info, ShouldNotBeNil)

				Convey("test lookup info", func() {
					So(info.OS.Company, ShouldResemble, "")
					So(info.OS.Family, ShouldResemble, "")
					So(info.OS.Icon, ShouldResemble, "")
					So(info.OS.Name, ShouldResemble, "")

					So(info.Device.Name, ShouldResemble, "Other")
					So(info.Device.Icon, ShouldResemble, "other.png")

					So(info.Browser.Company, ShouldResemble, "")
					So(info.Browser.Engine, ShouldResemble, "")
					So(info.Browser.Family, ShouldResemble, "")
					So(info.Browser.Icon, ShouldResemble, "")
					So(info.Browser.Name, ShouldResemble, "")
					So(info.Browser.Type, ShouldResemble, "Crawler")
					So(info.Browser.Version, ShouldResemble, "")

					So(info.Crawler.Name, ShouldResemble, "Googlebot Desktop")
					So(info.Crawler.Family, ShouldResemble, "Googlebot")
					So(info.Crawler.Vendor, ShouldResemble, "Google Inc.")
					So(info.Crawler.Class, ShouldResemble, "Search engine bot")
				})
			})
		})
	})
}

func TestRequiredLiterals(t *testing.T) {
	Convey("required literals ignore optional parts and alternations", t, func() {
		for regex, expected := range map[string][]string{
			`Linux.*?Android 11; (AFTN|Fire HD10)`: {"linux", "android 11; "},
			`^Mozilla/\d\.0 \(.*; Catchpoint\)`:    {"mozilla/", ".0 (", "; catchpoint)"},
			`(Opera)?Mini/([0-9.]+)`:               {"mini/"},
			`(abc){2}x?`:                           {"abc"},
		} {
			re, err := syntax.Parse("(?i)"+regex, syntax.Perl)
			So(err, ShouldBeNil)
			So(requiredLiterals(re), ShouldResemble, expected)
		}
	})
}

func BenchmarkUdger_Lookup(b *testing.B) {
	benchmarkLookup(b, []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2575.0 Safari/537.36"})
}

// benchmarkLookup reports the time taken by Lookup on uas, with and without the words prefilter
func benchmarkLookup(b *testing.B, uas []string) {
	for _, prefilter := range []bool{false, true} {
		b.Run(fmt.Sprintf("wordsPrefilter=%v", prefilter), func(b *testing.B) {
			udger, err := newUdger("./udgerdb_v3.dat", prefilter)
			if err != nil {
				panic(err)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err = udger.Lookup(uas[i%len(uas)])
				if err != nil {
					panic(err)
				}
			}
		})
	}
}

func TestWordsPrefilterLookup(t *testing.T) {
	Convey("Lookup gives the same result with and without the words prefilter", t, func() {
		fullScan, err := newUdger("./udgerdb_v3.dat", false)
		So(err, ShouldBeNil)
		prefiltered, err := New("./udgerdb_v3.dat")
		So(err, ShouldBeNil)

		for _, ua := range userAgents {
			expected, err := fullScan.Lookup(ua)
			So(err, ShouldBeNil)
			info, err := prefiltered.Lookup(ua)
			So(err, ShouldBeNil)
			So(info, ShouldResemble, expected)
		}
	})
}

// TestWordsPrefilterUserAgentsFile checks that Lookup gives the same result with and without the words prefilter on
// every user agent of user-agents.txt (one per line). It is skipped when the file is missing. Full scans are slow
// (~70 minutes for 500k user agents on 12 threads): run it with a larger timeout, e.g.
// go test -run TestWordsPrefilterUserAgentsFile -timeout 120m -v
// The logged lookup times are cumulated over concurrent workers: use the benchmarks to compare both modes.
func TestWordsPrefilterUserAgentsFile(t *testing.T) {
	file, err := os.Open("./user-agents.txt")
	if os.IsNotExist(err) {
		t.Skip("user-agents.txt not found")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var uas []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		uas = append(uas, strings.TrimSuffix(scanner.Text(), "\r"))
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	fullScan, err := newUdger("./udgerdb_v3.dat", false)
	if err != nil {
		t.Fatal(err)
	}
	prefiltered, err := New("./udgerdb_v3.dat")
	if err != nil {
		t.Fatal(err)
	}

	type mismatch struct {
		ua             string
		expected, info *Info
	}
	var (
		mu                            sync.Mutex
		mismatches                    []mismatch
		fullScanTime, prefilteredTime atomic.Int64
		wg                            sync.WaitGroup
	)
	jobs := make(chan string)
	for w := 0; w < runtime.GOMAXPROCS(0); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ua := range jobs {
				start := time.Now()
				expected, err := fullScan.Lookup(ua)
				fullScanTime.Add(int64(time.Since(start)))
				if err != nil {
					t.Errorf("full scan lookup of %q: %v", ua, err)
					continue
				}
				start = time.Now()
				info, err := prefiltered.Lookup(ua)
				prefilteredTime.Add(int64(time.Since(start)))
				if err != nil {
					t.Errorf("prefiltered lookup of %q: %v", ua, err)
					continue
				}
				if !reflect.DeepEqual(info, expected) {
					mu.Lock()
					mismatches = append(mismatches, mismatch{ua, expected, info})
					mu.Unlock()
				}
			}
		}()
	}
	for _, ua := range uas {
		jobs <- ua
	}
	close(jobs)
	wg.Wait()

	t.Logf("%d user agents, cumulated lookup time: full scan %v, words prefilter %v",
		len(uas), time.Duration(fullScanTime.Load()), time.Duration(prefilteredTime.Load()))
	for i, m := range mismatches {
		if i == 20 {
			t.Errorf("... %d more mismatches", len(mismatches)-i)
			break
		}
		t.Errorf("mismatch for %q:\nfull scan:       %+v\nwords prefilter: %+v", m.ua, *m.expected, *m.info)
	}
}

func TestWordsPrefilter(t *testing.T) {
	Convey("prefiltering regexes with words gives the same result as a full scan", t, func() {
		udger, err := New("./udgerdb_v3.dat")
		So(err, ShouldBeNil)

		uas := append([]string{""}, userAgents...)
		// a sample of crawlers, sorted to be deterministic: a full scan of all of them is too slow
		var crawlers []string
		for ua := range udger.Crawlers {
			crawlers = append(crawlers, ua)
		}
		sort.Strings(crawlers)
		for i := 0; i < len(crawlers); i += 20 {
			uas = append(uas, crawlers[i])
		}

		for _, ua := range uas {
			lowerUA := strings.ToLower(ua)
			for _, table := range []struct {
				rex         []rexData
				words       []rexWord
				withVersion bool
			}{
				{udger.rexBrowsers, udger.wordsBrowsers, true},
				{udger.rexOS, udger.wordsOS, false},
				{udger.rexDevices, udger.wordsDevices, false},
			} {
				fullID, fullVersion := udger.findDataWithVersion(ua, table.rex, nil, table.withVersion)
				id, version := udger.findDataWithVersion(ua, table.rex, udger.findWords(lowerUA, table.words), table.withVersion)
				So(id, ShouldEqual, fullID)
				So(version, ShouldEqual, fullVersion)
			}
		}
	})
}

// userAgents is a sample of common user agents: browsers, mobiles, smart TVs and libraries.
// TestWordsPrefilterLookup checks that each of them gives the same Lookup result with and without the words prefilter.
var userAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2575.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; x86_64 GNU/Linux) AppleWebKit/601.1 (KHTML, like Gecko) Version/8.0 Safari/601.1 WPE AAMP/4.6",
	"Mozilla/5.0 (SMART-TV; Linux; Tizen 6.0) AppleWebKit/538.1 (KHTML, like Gecko) Version/6.0 TV Safari/538.1",
	"Mozilla/5.0 (Web0S; Linux/SmartTV) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.270 Safari/537.36 WebAppManager",
	"Dalvik/2.1.0 (Linux; U; Android 11; BboxTV Build/RTT1.200907.001)",
	"Roku/DVP-15.3 (15.3.4.2402-AE)",
	"Apache-HttpClient/4.5.13 (Java/11.0.12)",
	// matches a Fire OS regex through "AFTN" while Udger tags it with the word "fire" from another alternation branch
	"XumoPlay/4.9.146 (Linux;Android 11; AFTNA1322D Build/RS8174.3641N)",
}

func BenchmarkUdger_LookupUserAgents(b *testing.B) {
	benchmarkLookup(b, userAgents)
}
