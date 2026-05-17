package hwprobe

import (
	"context"
	"testing"
)

func TestParseNvidiaSmiCSV_Multi(t *testing.T) {
	in := "NVIDIA RTX 4090, 24564\nNVIDIA RTX 3070, 8192\n"
	got := parseNvidiaSmiCSV(in)
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if got[0].Name != "NVIDIA RTX 4090" {
		t.Errorf("name[0]=%q", got[0].Name)
	}
	if got[0].VRAMBytes != 24564*1024*1024 {
		t.Errorf("vram[0]=%d", got[0].VRAMBytes)
	}
	if got[1].VRAMBytes != 8192*1024*1024 {
		t.Errorf("vram[1]=%d", got[1].VRAMBytes)
	}
}

func TestParseNvidiaSmiCSV_BadLinesSkipped(t *testing.T) {
	in := "garbage\nNVIDIA RTX 3070, 8192\nname-only\nNVIDIA, not-a-number\n"
	got := parseNvidiaSmiCSV(in)
	if len(got) != 1 || got[0].Name != "NVIDIA RTX 3070" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseNvidiaSmiCSV_Empty(t *testing.T) {
	if got := parseNvidiaSmiCSV(""); len(got) != 0 {
		t.Errorf("empty: got %#v", got)
	}
	if got := parseNvidiaSmiCSV("\n   \n"); len(got) != 0 {
		t.Errorf("whitespace: got %#v", got)
	}
}

func TestSystemProbe_InjectedReaders(t *testing.T) {
	p := &SystemProbe{
		RAMReader: func() (uint64, error) { return 16 * 1024 * 1024 * 1024, nil },
		GPUReader: func(context.Context) ([]GPU, error) {
			return []GPU{{Name: "Fake", VRAMBytes: 8 << 30}}, nil
		},
		CPUCount: func() int { return 8 },
	}
	caps, err := p.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if caps.CPUCores != 8 || caps.TotalRAMBytes != 16<<30 || len(caps.GPUs) != 1 {
		t.Fatalf("caps=%#v", caps)
	}
}
