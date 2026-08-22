package redisstate

import "testing"

// Store selection decides whether revocation state is shared across replicas,
// so it is driven by explicit configuration rather than by whether a server
// happens to answer. A probe-based selector would silently downgrade a
// multi-replica deployment to a per-process blacklist during a Redis outage,
// which is exactly when revocation matters most.
func TestRedisConfiguredReadsDeclaredConfigurationOnly(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{name: "no redis environment", env: map[string]string{}, want: false},
		{name: "url declared", env: map[string]string{"REDIS_URL": "redis://localhost:6379"}, want: true},
		{name: "host declared", env: map[string]string{"REDIS_HOST": "localhost"}, want: true},
		{name: "blank url is not configuration", env: map[string]string{"REDIS_URL": "   "}, want: false},
		{name: "port without host is not configuration", env: map[string]string{"REDIS_PORT": "6379"}, want: false},
		{name: "unreachable host is still configuration", env: map[string]string{"REDIS_HOST": "203.0.113.1"}, want: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			for _, name := range []string{"REDIS_URL", "REDIS_HOST", "REDIS_PORT", "REDIS_DB", "REDIS_PASSWORD"} {
				t.Setenv(name, "")
			}
			for name, value := range testCase.env {
				t.Setenv(name, value)
			}
			if got := RedisConfigured(); got != testCase.want {
				t.Fatalf("RedisConfigured() = %v, want %v", got, testCase.want)
			}
		})
	}
}
