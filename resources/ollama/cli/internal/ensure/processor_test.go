package ensure

import "testing"

func TestSummarizeProcessors(t *testing.T) {
	cases := []struct {
		name string
		in   []RunningModel
		want ProcessorState
		cpu  bool
		gpu  bool
	}{
		{"gpu", []RunningModel{{Name: "a", Processor: "100% GPU"}}, ProcessorGPU, false, true},
		{"gpu from vram", []RunningModel{{Name: "a", Size: 100, SizeVRAM: 90}}, ProcessorGPU, false, true},
		{"cpu", []RunningModel{{Name: "a", Processor: "100% CPU"}}, ProcessorCPU, true, false},
		{"cpu from zero vram", []RunningModel{{Name: "a", Size: 100}}, ProcessorCPU, true, false},
		{"mixed", []RunningModel{{Name: "a", Processor: "100% GPU"}, {Name: "b", Processor: "100% CPU"}}, ProcessorState("mixed"), true, true},
		{"empty", nil, ProcessorUnknown, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeProcessors(tc.in)
			if got.Processor != tc.want || got.HasCPUModel != tc.cpu || got.HasGPUModel != tc.gpu {
				t.Fatalf("report=%+v, want processor=%q cpu=%v gpu=%v", got, tc.want, tc.cpu, tc.gpu)
			}
		})
	}
}
