// tsvconv converts output of benchstat to tab separated values.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	pkgName  = regexp.MustCompile(`/(?P<pkg>[a-zA-Z]+)\s+`)
	dataLine = regexp.MustCompile(`^(?P<name>.+)-(?P<cpus>\d+)\s*(?P<val>\d+(?:\.\d+)?)(?P<unit>[µmn])s\s*±\s*(?P<error>\d+)%`)

	dataNames map[string]int
	alldata   map[string]map[string][]coldata
)

func init() {
	dataNames = make(map[string]int)
	for i, n := range dataLine.SubexpNames() {
		dataNames[n] = i
	}
	alldata = make(map[string]map[string][]coldata)
}

type coldata struct {
	coltitle string
	value    float64
	below    float64
	above    float64
	original string
}

func main() {
	flag.Parse()
	f, err := os.OpenFile(flag.Arg(0), os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	lines := bytes.Split(b, []byte("\n"))
	var title string
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("pkg:")) {
			title = string(pkgName.FindSubmatch(line)[1])
			alldata[title] = make(map[string][]coldata)
		} else {
			matches := dataLine.FindSubmatch(line)
			if len(matches) > 2 {
				name := string(matches[dataNames["name"]])
				index := strings.Index(name, "/")
				rowtitle := name
				coltitle := "default" // default parameters
				if index != -1 {
					rowtitle = name[:index]
					coltitle = name[index+1:]
				}
				if _, ok := alldata[title][rowtitle]; !ok {
					if _, ok := alldata[title]; !ok {
						alldata[title] = make(map[string][]coldata)
					}
					alldata[title][rowtitle] = make([]coldata, 0)
				}
				unit := string(matches[dataNames["unit"]])
				vals := string(matches[dataNames["val"]])
				value, err := strconv.ParseFloat(vals, 32)
				if err != nil {
					log.Fatalf("could not parse value: %s", string(line))
				}
				if unit == "m" {
					value = value * 1000
				} else if unit == "n" {
					value = value / 1000
				} else { // micro
				}
				errs := string(matches[dataNames["error"]])
				errval, err := strconv.ParseFloat(errs, 32)
				if err != nil {
					log.Fatalf("could not parse error: %s", string(line))
				}
				delta := (errval / 100)
				row := coldata{
					coltitle,
					value,
					value * (1 + delta),
					value * (1 - delta),
					string(line),
				}
				alldata[title][rowtitle] = append(alldata[title][rowtitle], row)
			} else {
				log.Println("? ", string(line))
			}
		}
	}
	for title, row := range alldata {
		fmt.Printf("# ---- %s ----\n", title)
		for _, data := range row {
			fmt.Printf("#\t")
			for _, colgroup := range data {
				fmt.Printf("%s\t+err\t-err\t", colgroup.coltitle)
			}
			fmt.Println()
			break
		}
		for col, data := range row {
			fmt.Printf("%s\t", col)
			for _, colgroup := range data {
				fmt.Printf("%.2f\t%.2f\t%.2f\t", colgroup.value, colgroup.above, colgroup.below)
			}
			fmt.Printf("\n")
		}
		fmt.Println()
	}
}
