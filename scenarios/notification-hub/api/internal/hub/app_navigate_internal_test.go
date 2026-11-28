package hub

import "testing"

func TestAppNavigateURLKeepsTheTapInsideTheApp(t *testing.T) {
	const origin = "https://web-console.example.test"
	cases := map[string]string{
		"":                                  origin + "/",
		"/sessions/abc?tab=2":               origin + "/sessions/abc?tab=2",
		origin + "/sessions/abc#reply":      origin + "/sessions/abc",
		"https://WEB-console.example.test/": origin + "/",
		"https://agent-manager.example.test/runs/1": origin + "/",
		"//evil.example.test/x":                     origin + "/",
	}
	for contextURL, want := range cases {
		if got := appNavigateURL(origin+"/", contextURL); got != want {
			t.Errorf("appNavigateURL(%q) = %q, want %q", contextURL, got, want)
		}
	}
}

func TestDeviceLabelNamesDevicesForPeople(t *testing.T) {
	cases := []struct{ ua, endpoint, want string }{
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X) Safari/604.1", "", "iPhone"},
		{"Mozilla/5.0 (Linux; Android 15) Chrome/141.0 Mobile Safari/537.36", "", "Android phone"},
		{"Mozilla/5.0 (Windows NT 10.0) Chrome/141.0 Safari/537.36 Edg/141.0", "", "Windows · Edge"},
		{"", "https://web.push.apple.com/abc", "Apple device"},
		{"", "https://fcm.googleapis.com/fcm/send/abc", "Chrome or Android device"},
	}
	for _, c := range cases {
		if got := deviceLabel(c.ua, c.endpoint); got != c.want {
			t.Errorf("deviceLabel(%q, %q) = %q, want %q", c.ua, c.endpoint, got, c.want)
		}
	}
}
