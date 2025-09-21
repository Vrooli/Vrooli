package main

import "strings"

func containsViolationSubstring(list []Violation, needle string) bool {
	for _, v := range list {
		if strings.Contains(v.Message, needle) {
			return true
		}
	}
	return false
}
