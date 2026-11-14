package usagelabel

import "testing"

func TestTitlesFollowTheNamingRule(t *testing.T) {
	cases := []struct {
		provider, window string
		minutes          int64
		want             string
	}{
		{"anthropic", "10080m", 10080, "Claude · weekly"},
		{"openai", "300m", 300, "Codex · 5-hour"},
		{"openai", "seven_day", 0, "Codex · weekly"},
		{"anthropic", "five_hour", 0, "Claude · 5-hour"},
		{"anthropic", "1440m", 1440, "Claude · daily"},
		{"anthropic", "300m", 0, "Claude · 5-hour"},
		{"openai", "299m", 299, "Codex · 5-hour"},
		{"openai", "301m", 301, "Codex · 5-hour"},
		{"openai", "299m", 0, "Codex · 5-hour"},
		{"openai", "10079m", 10079, "Codex · weekly"},
		{"openai", "10081m", 10081, "Codex · weekly"},
		{"openai", "10077m", 10077, "Codex · 10077m"},
		{"openai", "290m", 290, "Codex · 290m"},
	}
	for _, c := range cases {
		if got := Title(c.provider, c.window, c.minutes); got != c.want {
			t.Errorf("Title(%q, %q, %d) = %q, want %q", c.provider, c.window, c.minutes, got, c.want)
		}
	}
}

func TestMinutesRoundsToTheNearestWholeHourWithinTwoMinutes(t *testing.T) {
	for in, want := range map[int64]int64{299: 300, 300: 300, 301: 300, 10079: 10080, 10080: 10080, 10078: 10080, 10077: 10077, 45: 45, 1439: 1440} {
		if got := Minutes(in, ""); got != want {
			t.Errorf("Minutes(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestTokensAndDurationReadAsOperatorsSayThem(t *testing.T) {
	for in, want := range map[int64]string{950: "950", 12500: "12.5k", 3_000_000: "3.0M"} {
		if got := Tokens(in); got != want {
			t.Errorf("Tokens(%d) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[int64]string{1500: "1.5 s", 185_000: "3m 05s", 7_440_000: "2h 04m"} {
		if got := Duration(in); got != want {
			t.Errorf("Duration(%d) = %q, want %q", in, got, want)
		}
	}
}
