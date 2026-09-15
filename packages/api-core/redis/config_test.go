package redis

import "testing"

func TestResolveURLAndComponents(t *testing.T) {
	values := map[string]string{"REDIS_URL": "redis://:secret@redis.example:6380/4"}
	cfg, err := Resolve(func(key string) string { return values[key] })
	if err != nil || cfg.Addr != "redis.example:6380" || cfg.Password != "secret" || cfg.DB != 4 {
		t.Fatalf("cfg=%#v err=%v", cfg, err)
	}
	values = map[string]string{"REDIS_HOST": "redis", "REDIS_PORT": "6379", "REDIS_DB": "2", "REDIS_PASSWORD": "pw"}
	cfg, err = Resolve(func(key string) string { return values[key] })
	if err != nil || cfg.Addr != "redis:6379" || cfg.Password != "pw" || cfg.DB != 2 {
		t.Fatalf("cfg=%#v err=%v", cfg, err)
	}
}
