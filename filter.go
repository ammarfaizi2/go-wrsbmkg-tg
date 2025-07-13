package main

import "strings"

func checkFilter(s string) (pass bool) {
	if len(config.FilterRegions) == 0 { // if user didn't set filter...
		pass = true
		return
	}

	s = strings.ToLower(s)

	pass = false // we do not give it a pass until one of the filter is on the list

	for _, f := range config.FilterRegions {
		if strings.Contains(s, f) {
			pass = true
			return
		}
	}

	return
}
