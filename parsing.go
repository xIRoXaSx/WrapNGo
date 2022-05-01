package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

func parseDate(tm time.Time, format string) (date string, err error) {
	date = format
	formats := map[string]string{
		"YYYY": fmt.Sprintf("%d", tm.Year()),
		"YYY":  fmt.Sprintf("%d", tm.YearDay()),
		"YY":   fmt.Sprintf("%d", tm.Year())[2:],
		"MMMM": tm.Month().String(),
		"MMM":  tm.Month().String()[:3],
		"MM":   fmt.Sprintf("%02d", int(tm.Month())),
		"M":    fmt.Sprintf("%d", int(tm.Month())),
		"DDDD": tm.Weekday().String(),
		"DDD":  tm.Weekday().String()[:3],
		"DD":   fmt.Sprintf("%02d", tm.Day()),
		"D":    fmt.Sprintf("%d", tm.Day()),
		"hha":  tm.Format(time.Kitchen),
		"hh":   fmt.Sprintf("%02d", tm.Hour()),
		"h":    fmt.Sprintf("%d", tm.Hour()),
		"mm":   fmt.Sprintf("%02d", tm.Minute()),
		"m":    fmt.Sprintf("%02d", tm.Minute()),
		"ss":   fmt.Sprintf("%02d", tm.Second()),
		"s":    fmt.Sprintf("%d", tm.Second()),
		"ms":   fmt.Sprintf("%d", int32(tm.Nanosecond())/int32(time.Millisecond)),
	}
	// Values need to be ordered to get consistent and correct results.
	ordered := []string{
		"YYYY", "YYY", "YY", "MMMM", "MMM", "MM", "M", "DDDD", "DDD", "DD", "D",
		"hha", "hh", "h", "mm", "m", "ss", "s", "ms",
	}

	var reg *regexp.Regexp
	for _, f := range ordered {
		reg, err = regexp.Compile(fmt.Sprintf("(%s)", regexp.QuoteMeta(f)))
		if err != nil {
			return
		}

		match := reg.FindStringSubmatch(f)
		if len(match) > 0 {
			date = strings.ReplaceAll(date, match[1], formats[f])
		}
	}
	return
}
