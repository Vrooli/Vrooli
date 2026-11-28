package worktree

import "testing"

// TestParsePorcelain exercises the worktree-list parser against fixed
// strings. NO real git is invoked — these inputs are hand-crafted to
// match the porcelain format git emits.
func TestParsePorcelain(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Worktree
	}{
		{
			name: "main only",
			in: `worktree /repo
HEAD aaaa
branch refs/heads/main

`,
			want: []Worktree{
				{Path: "/repo", Name: "repo", HeadCommit: "aaaa", Branch: "main", IsMain: true},
			},
		},
		{
			name: "main + linked",
			in: `worktree /repo
HEAD aaaa
branch refs/heads/main

worktree /tmp/feature
HEAD bbbb
branch refs/heads/feature

`,
			want: []Worktree{
				{Path: "/repo", Name: "repo", HeadCommit: "aaaa", Branch: "main", IsMain: true},
				{Path: "/tmp/feature", Name: "feature", HeadCommit: "bbbb", Branch: "feature"},
			},
		},
		{
			name: "detached + locked + prunable",
			in: `worktree /repo
HEAD aaaa
branch refs/heads/main

worktree /tmp/detached
HEAD bbbb
detached

worktree /tmp/locked
HEAD cccc
branch refs/heads/staging
locked agents pinned

worktree /tmp/gone
HEAD dddd
branch refs/heads/orphan
prunable gitdir file points to non-existent location
`,
			want: []Worktree{
				{Path: "/repo", Name: "repo", HeadCommit: "aaaa", Branch: "main", IsMain: true},
				{Path: "/tmp/detached", Name: "detached", HeadCommit: "bbbb", Detached: true},
				{Path: "/tmp/locked", Name: "locked", HeadCommit: "cccc", Branch: "staging", Locked: true, LockReason: "agents pinned"},
				{Path: "/tmp/gone", Name: "gone", HeadCommit: "dddd", Branch: "orphan", Prunable: true, PrunableReason: "gitdir file points to non-existent location"},
			},
		},
		{
			name: "empty input",
			in:   ``,
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parsePorcelain(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("len: got %d want %d (%+v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d]:\n got:  %+v\n want: %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}
