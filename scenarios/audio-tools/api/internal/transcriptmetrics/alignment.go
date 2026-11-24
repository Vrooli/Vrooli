package transcriptmetrics

// EditCounts is the substitution/insertion/deletion breakdown of one
// alignment.
type EditCounts struct {
	Substitutions int
	Insertions    int
	Deletions     int
}

// EditOperation is one step in the aligned reference/hypothesis token path.
type EditOperation struct {
	Kind            string
	ReferenceToken  string
	HypothesisToken string
	ReferenceIndex  int
	HypothesisIndex int
}

// Total is the raw edit distance S+I+D.
func (e EditCounts) Total() int { return e.Substitutions + e.Insertions + e.Deletions }

// WERResult is an alignment plus the sequence lengths needed to interpret it.
type WERResult struct {
	EditCounts
	RefWords int
	HypWords int
}

// Rate is (S+I+D)/N_ref. Empty reference and hypothesis is perfect; a
// non-empty hypothesis against an empty reference has rate 1.
func (w WERResult) Rate() float64 {
	if w.RefWords == 0 {
		if w.HypWords == 0 {
			return 0
		}
		return 1
	}
	return float64(w.Total()) / float64(w.RefWords)
}

// Align computes the minimum-edit alignment between two token sequences.
// Equal-cost paths break ties in substitution, deletion, insertion order.
func Align(ref, hyp []string) EditCounts {
	return alignCounts(ref, hyp)
}

// alignCounts computes the same minimum edit counts as alignPath while keeping
// only two rows. The traceback matrix remains exclusive to AlignOperations.
func alignCounts(ref, hyp []string) EditCounts {
	previous := make([]EditCounts, len(hyp)+1)
	for j := 1; j <= len(hyp); j++ {
		previous[j] = EditCounts{Insertions: j}
	}
	current := make([]EditCounts, len(hyp)+1)
	for i, refToken := range ref {
		current[0] = EditCounts{Deletions: i + 1}
		for j, hypToken := range hyp {
			if refToken == hypToken {
				current[j+1] = previous[j]
				continue
			}
			sub := incSub(previous[j])
			del := incDel(previous[j+1])
			ins := current[j]
			ins.Insertions++
			current[j+1] = minCounts(sub, del, ins)
		}
		previous, current = current, previous
	}
	return previous[len(hyp)]
}

// AlignOperations returns the deterministic alignment and its operation path.
func AlignOperations(ref, hyp []string) (EditCounts, []EditOperation) {
	path := alignPath(ref, hyp, true)
	return path.counts, path.ops
}

// WER aligns tokenized reference and hypothesis sequences.
func WER(ref, hyp []string) WERResult {
	return WERResult{EditCounts: Align(ref, hyp), RefWords: len(ref), HypWords: len(hyp)}
}

type alignmentPath struct {
	counts EditCounts
	ops    []EditOperation
}

func alignPath(ref, hyp []string, keepOps bool) alignmentPath {
	n, m := len(ref), len(hyp)
	d := make([][]EditCounts, n+1)
	for i := range d {
		d[i] = make([]EditCounts, m+1)
	}
	for i := 1; i <= n; i++ {
		d[i][0] = EditCounts{Deletions: i}
	}
	for j := 1; j <= m; j++ {
		d[0][j] = EditCounts{Insertions: j}
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if ref[i-1] == hyp[j-1] {
				d[i][j] = d[i-1][j-1]
				continue
			}
			sub := d[i-1][j-1]
			sub.Substitutions++
			del := d[i-1][j]
			del.Deletions++
			ins := d[i][j-1]
			ins.Insertions++
			d[i][j] = minCounts(sub, del, ins)
		}
	}
	out := alignmentPath{counts: d[n][m]}
	if !keepOps {
		return out
	}
	i, j := n, m
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && ref[i-1] == hyp[j-1] && d[i][j] == d[i-1][j-1]:
			out.ops = append(out.ops, EditOperation{Kind: "match", ReferenceToken: ref[i-1], HypothesisToken: hyp[j-1], ReferenceIndex: i - 1, HypothesisIndex: j - 1})
			i--
			j--
		case i > 0 && j > 0 && editCountsEqual(d[i][j], incSub(d[i-1][j-1])):
			out.ops = append(out.ops, EditOperation{Kind: "substitution", ReferenceToken: ref[i-1], HypothesisToken: hyp[j-1], ReferenceIndex: i - 1, HypothesisIndex: j - 1})
			i--
			j--
		case i > 0 && editCountsEqual(d[i][j], incDel(d[i-1][j])):
			out.ops = append(out.ops, EditOperation{Kind: "deletion", ReferenceToken: ref[i-1], ReferenceIndex: i - 1, HypothesisIndex: -1})
			i--
		default:
			out.ops = append(out.ops, EditOperation{Kind: "insertion", HypothesisToken: hyp[j-1], ReferenceIndex: -1, HypothesisIndex: j - 1})
			j--
		}
	}
	for left, right := 0, len(out.ops)-1; left < right; left, right = left+1, right-1 {
		out.ops[left], out.ops[right] = out.ops[right], out.ops[left]
	}
	return out
}

func minCounts(a, b, c EditCounts) EditCounts {
	best := a
	if b.Total() < best.Total() {
		best = b
	}
	if c.Total() < best.Total() {
		best = c
	}
	return best
}

func incSub(e EditCounts) EditCounts { e.Substitutions++; return e }
func incDel(e EditCounts) EditCounts { e.Deletions++; return e }
func editCountsEqual(a, b EditCounts) bool {
	return a.Substitutions == b.Substitutions && a.Insertions == b.Insertions && a.Deletions == b.Deletions
}
