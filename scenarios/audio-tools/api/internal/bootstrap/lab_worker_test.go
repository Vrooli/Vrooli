package bootstrap

import "testing"

func TestLabWorkerRequiresLabVariantAndExperimentService(t *testing.T) {
	for _, tc := range []struct {
		name       string
		labEnabled bool
		service    bool
		want       bool
	}{
		{name: "product with service", service: true},
		{name: "lab without service", labEnabled: true},
		{name: "lab with service", labEnabled: true, service: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldStartLabWorker(tc.labEnabled, tc.service); got != tc.want {
				t.Fatalf("shouldStartLabWorker(%v, %v) = %v, want %v", tc.labEnabled, tc.service, got, tc.want)
			}
		})
	}
}
