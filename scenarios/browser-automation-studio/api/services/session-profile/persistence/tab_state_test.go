package persistence

import "testing"

func TestLimitRestoredTabsKeepsFirstTwentyInCallerOrder(t *testing.T) {
	tabs := make([]TabState, MaxRestoredTabs+3)
	for i := range tabs {
		tabs[i] = TabState{URL: "https://example.test/" + string(rune('a'+i)), Order: i}
	}
	limited := LimitRestoredTabs(tabs)
	if len(limited) != MaxRestoredTabs || limited[0] != tabs[0] || limited[len(limited)-1] != tabs[MaxRestoredTabs-1] {
		t.Fatalf("limited values do not preserve the first %d tabs: len=%d first=%+v last=%+v", MaxRestoredTabs, len(limited), limited[0], limited[len(limited)-1])
	}
	if len(tabs) != MaxRestoredTabs+3 {
		t.Fatal("limiting tabs mutated the caller slice")
	}
	if got := LimitRestoredTabs(nil); got != nil {
		t.Fatal("nil tab collection changed")
	}
}
