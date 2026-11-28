package domain

import "testing"

type enumPolicyCase[T ~string, R comparable] struct {
	value T
	want  R
}

func assertEnumPolicy[T ~string, R comparable](t *testing.T, cases []enumPolicyCase[T, R], evaluate func(T) R) {
	t.Helper()
	for _, tt := range cases {
		t.Run(string(tt.value), func(t *testing.T) {
			if got := evaluate(tt.value); got != tt.want {
				t.Errorf("policy result = %v, want %v", got, tt.want)
			}
		})
	}
}
