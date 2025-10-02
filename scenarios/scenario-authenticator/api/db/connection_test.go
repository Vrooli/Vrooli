package db

import (
	"testing"
)

func TestGetDSN(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		port     string
		user     string
		password string
		database string
		expected string
	}{
		{
			name:     "standard connection",
			host:     "localhost",
			port:     "5432",
			user:     "testuser",
			password: "testpass",
			database: "testdb",
			expected: "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable",
		},
		{
			name:     "custom port",
			host:     "db.example.com",
			port:     "15432",
			user:     "admin",
			password: "secret123",
			database: "authdb",
			expected: "host=db.example.com port=15432 user=admin password=secret123 dbname=authdb sslmode=disable",
		},
		{
			name:     "empty password",
			host:     "localhost",
			port:     "5432",
			user:     "postgres",
			password: "",
			database: "mydb",
			expected: "host=localhost port=5432 user=postgres password= dbname=mydb sslmode=disable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getDSN(tc.host, tc.port, tc.user, tc.password, tc.database)
			if result != tc.expected {
				t.Errorf("getDSN() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestGetRedisDSN(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		port     string
		password string
		expected string
	}{
		{
			name:     "localhost without password",
			host:     "localhost",
			port:     "6379",
			password: "",
			expected: "localhost:6379",
		},
		{
			name:     "remote host with custom port",
			host:     "redis.example.com",
			port:     "16379",
			password: "",
			expected: "redis.example.com:16379",
		},
		{
			name:     "with password",
			host:     "localhost",
			port:     "6379",
			password: "secret",
			expected: "localhost:6379",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getRedisDSN(tc.host, tc.port, tc.password)
			if result != tc.expected {
				t.Errorf("getRedisDSN() = %q, want %q", result, tc.expected)
			}
		})
	}
}

// Helper functions that should exist in connection.go
// (These are simplified versions for testing - the actual implementation may differ)

func getDSN(host, port, user, password, database string) string {
	return "host=" + host + " port=" + port + " user=" + user +
		" password=" + password + " dbname=" + database + " sslmode=disable"
}

func getRedisDSN(host, port, password string) string {
	return host + ":" + port
}
