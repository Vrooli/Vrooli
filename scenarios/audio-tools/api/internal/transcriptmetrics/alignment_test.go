package transcriptmetrics

import "testing"

func TestAlignCountsForEmptyAndMixedEdits(t *testing.T) {
	tests := []struct {
		name     string
		ref, hyp []string
		want     EditCounts
	}{
		{name: "both empty"},
		{name: "reference empty", hyp: []string{"a", "b"}, want: EditCounts{Insertions: 2}},
		{name: "hypothesis empty", ref: []string{"a", "b"}, want: EditCounts{Deletions: 2}},
		{name: "mixed edits", ref: []string{"a", "b", "c"}, hyp: []string{"a", "x", "c", "d"}, want: EditCounts{Substitutions: 1, Insertions: 1}},
		{name: "duplicate tokens", ref: []string{"a", "a"}, hyp: []string{"a"}, want: EditCounts{Deletions: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Align(tt.ref, tt.hyp); got != tt.want {
				t.Fatalf("Align() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAlignOperationsDeterministicTieBreakAndIndices(t *testing.T) {
	counts, ops := AlignOperations([]string{"a"}, []string{"b", "a"})
	if counts != (EditCounts{Insertions: 1}) {
		t.Fatalf("counts = %+v", counts)
	}
	want := []EditOperation{{Kind: "insertion", HypothesisToken: "b", ReferenceIndex: -1, HypothesisIndex: 0}, {Kind: "match", ReferenceToken: "a", HypothesisToken: "a", ReferenceIndex: 0, HypothesisIndex: 1}}
	if len(ops) != len(want) {
		t.Fatalf("ops = %#v", ops)
	}
	for i := range want {
		if ops[i] != want[i] {
			t.Errorf("ops[%d] = %+v, want %+v", i, ops[i], want[i])
		}
	}
	_, tied := AlignOperations([]string{"a"}, []string{"b"})
	if len(tied) != 1 || tied[0].Kind != "substitution" {
		t.Fatalf("equal-cost tie must prefer substitution: %+v", tied)
	}
}

func TestWERRateDefinesEmptyReference(t *testing.T) {
	if got := WER(nil, nil).Rate(); got != 0 {
		t.Fatalf("empty/empty rate = %v", got)
	}
	if got := WER(nil, []string{"word"}).Rate(); got != 1 {
		t.Fatalf("empty reference rate = %v", got)
	}
	result := WER([]string{"a", "b"}, []string{"a", "x", "c"})
	if result.RefWords != 2 || result.HypWords != 3 || result.Total() != 2 || result.Rate() != 1 {
		t.Fatalf("unexpected WER result: %+v, rate=%v", result, result.Rate())
	}
}

func TestAlignCountsMatchesTracebackAcrossTokenSequences(t *testing.T) {
	alphabet := []string{"a", "b"}
	sequences := [][]string{{}}
	for length := 1; length <= 5; length++ {
		for mask := 0; mask < 1<<length; mask++ {
			seq := make([]string, length)
			for i := range seq {
				if mask&(1<<i) != 0 {
					seq[i] = alphabet[1]
				} else {
					seq[i] = alphabet[0]
				}
			}
			sequences = append(sequences, seq)
		}
	}
	for _, ref := range sequences {
		for _, hyp := range sequences {
			want := alignPath(ref, hyp, false).counts
			if got := Align(ref, hyp); got != want {
				t.Fatalf("Align(%v, %v) = %+v, matrix = %+v", ref, hyp, got, want)
			}
		}
	}
}

func BenchmarkAlignLongSequence(b *testing.B) {
	ref, hyp := make([]string, 800), make([]string, 800)
	for i := range ref {
		ref[i] = "word"
		hyp[i] = "word"
	}
	hyp[400] = "changed"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Align(ref, hyp)
	}
}
