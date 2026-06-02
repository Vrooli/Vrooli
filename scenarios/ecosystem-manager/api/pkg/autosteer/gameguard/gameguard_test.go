package gameguard

import (
	"reflect"
	"sort"
	"testing"
)

func causes(cs ...Cause) []Cause {
	sort.Slice(cs, func(i, j int) bool { return cs[i] < cs[j] })
	return cs
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name        string
		diff        Diff
		wantGamed   bool
		wantCauses  []Cause
		wantFlagged bool
	}{
		{
			name: "test-weakening on a [REQ:] test",
			diff: Diff{Content: `diff --git a/api/health_test.go b/api/health_test.go
--- a/api/health_test.go
+++ b/api/health_test.go
@@ -10,7 +10,7 @@
 // [REQ:HEALTH-1] health endpoint returns ok
-	if resp.Status != "ok" {
-		t.Fatalf("want ok, got %s", resp.Status)
-	}
+	if resp.Status == "" {
+		t.Log("status present")
+	}
`},
			wantGamed:  true,
			wantCauses: causes(CauseTestWeakening),
		},
		{
			name: "ledger deletion via content marker",
			diff: Diff{Content: `diff --git a/PROBLEMS.md b/PROBLEMS.md
deleted file mode 100644
--- a/PROBLEMS.md
+++ /dev/null
@@ -1,3 +0,0 @@
-# Known Problems
-- missing export feature
`},
			wantGamed:  true,
			wantCauses: causes(CauseLedgerDeletion),
		},
		{
			name: "ledger deletion via file record only",
			diff: Diff{
				Content: "",
				Files:   []FileChange{{Path: "docs/PROGRESS.md", ChangeType: "deleted", Deletions: 12}},
			},
			wantGamed:  true,
			wantCauses: causes(CauseLedgerDeletion),
		},
		{
			name: "suppression: nolint added",
			diff: Diff{Content: `diff --git a/api/x.go b/api/x.go
--- a/api/x.go
+++ b/api/x.go
@@ -5,3 +5,4 @@
 func x() {
+	//nolint:errcheck
 	doThing()
`},
			wantGamed:  true,
			wantCauses: causes(CauseSuppression),
		},
		{
			name: "suppression: eslint-disable added",
			diff: Diff{Content: `diff --git a/ui/src/a.ts b/ui/src/a.ts
--- a/ui/src/a.ts
+++ b/ui/src/a.ts
@@ -1,2 +1,3 @@
+// eslint-disable-next-line @typescript-eslint/no-explicit-any
 const x: any = y
`},
			wantGamed:  true,
			wantCauses: causes(CauseSuppression),
		},
		{
			name: "suppression: rule disabled in auditor config",
			diff: Diff{Content: `diff --git a/.vrooli/testing.json b/.vrooli/testing.json
--- a/.vrooli/testing.json
+++ b/.vrooli/testing.json
@@ -3,3 +3,4 @@
   "rule": "type-safety",
+  "enabled": false
`},
			wantGamed:  true,
			wantCauses: causes(CauseSuppression),
		},
		{
			name: "ambiguous: assertions removed from non-REQ test",
			diff: Diff{Content: `diff --git a/api/util_test.go b/api/util_test.go
--- a/api/util_test.go
+++ b/api/util_test.go
@@ -10,5 +10,3 @@
-	if got != want {
-		t.Errorf("bad")
-	}
+	_ = got
`},
			wantGamed:   false,
			wantCauses:  nil,
			wantFlagged: true,
		},
		{
			name: "clean substantive control — adds code AND tests",
			diff: Diff{Content: `diff --git a/api/export.go b/api/export.go
--- a/api/export.go
+++ b/api/export.go
@@ -1,3 +1,8 @@
 package api
+
+func Export() error {
+	return nil
+}
diff --git a/api/export_test.go b/api/export_test.go
--- a/api/export_test.go
+++ b/api/export_test.go
@@ -1,2 +1,7 @@
 package api
+
+func TestExport(t *testing.T) {
+	if err := Export(); err != nil {
+		t.Fatalf("want nil, got %v", err)
+	}
+}
`},
			wantGamed:   false,
			wantCauses:  nil,
			wantFlagged: false,
		},
		{
			name: "multiple causes",
			diff: Diff{Content: `diff --git a/PROBLEMS.md b/PROBLEMS.md
deleted file mode 100644
--- a/PROBLEMS.md
+++ /dev/null
@@ -1,1 +0,0 @@
-- gap
diff --git a/api/x.go b/api/x.go
--- a/api/x.go
+++ b/api/x.go
@@ -1,1 +1,2 @@
+	//nolint:all
`},
			wantGamed:  true,
			wantCauses: causes(CauseLedgerDeletion, CauseSuppression),
		},
		{
			name:        "empty diff is clean",
			diff:        Diff{},
			wantGamed:   false,
			wantCauses:  nil,
			wantFlagged: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.diff)
			if got.Gamed != tc.wantGamed {
				t.Errorf("Gamed = %v, want %v (details: %v)", got.Gamed, tc.wantGamed, got.Details)
			}
			if !reflect.DeepEqual(nilIfEmpty(got.Causes), nilIfEmpty(tc.wantCauses)) {
				t.Errorf("Causes = %v, want %v", got.Causes, tc.wantCauses)
			}
			if got.FlaggedForReview != tc.wantFlagged {
				t.Errorf("FlaggedForReview = %v, want %v (details: %v)", got.FlaggedForReview, tc.wantFlagged, got.Details)
			}
		})
	}
}

func nilIfEmpty(cs []Cause) []Cause {
	if len(cs) == 0 {
		return nil
	}
	return cs
}

func TestCauseString(t *testing.T) {
	r := Result{Causes: []Cause{CauseLedgerDeletion, CauseSuppression}}
	if got := r.CauseString(); got != "gamed:ledger-deletion,suppression" {
		t.Errorf("CauseString = %q", got)
	}
	if (Result{}).CauseString() != "" {
		t.Error("empty result should yield empty cause string")
	}
}
