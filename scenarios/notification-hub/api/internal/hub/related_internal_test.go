package hub

import "testing"

func TestLexicalScoreCalibration(t *testing.T) {
	a := relatedWords("Should the agent delete the staging database?")
	b := relatedWords("Should the agent be allowed to drop the staging DB?")
	c := relatedWords("Which logo variant should the landing page use?")
	t.Logf("a=%v b=%v", a, b)
	t.Logf("ab=%v ac=%v", lexicalScore(a, b), lexicalScore(a, c))
	if lexicalScore(a, b) < relatedFloor {
		t.Fatalf("similar pair scored %v below floor %v", lexicalScore(a, b), relatedFloor)
	}
	if lexicalScore(a, c) >= relatedFloor {
		t.Fatalf("unrelated pair scored %v above floor", lexicalScore(a, c))
	}
}
