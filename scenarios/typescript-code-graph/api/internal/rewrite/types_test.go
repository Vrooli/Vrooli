package rewrite

import "testing"

func TestOperationTag(t *testing.T) {
	cases := []struct {
		name string
		op   Operation
		want string
	}{
		{"file_move", Operation{FileMove: &FileMove{FromPath: "a", ToPath: "b"}}, "file_move"},
		{"import_rewrite", Operation{ImportRewrite: &ImportRewrite{OldPath: "a", NewPath: "b"}}, "import_rewrite"},
		{"empty", Operation{}, ""},
		{"both", Operation{FileMove: &FileMove{}, ImportRewrite: &ImportRewrite{}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.op.OperationTag(); got != tc.want {
				t.Errorf("OperationTag()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestPrimaryAndSecondaryPath(t *testing.T) {
	fm := Operation{FileMove: &FileMove{FromPath: "from", ToPath: "to"}}
	if fm.PrimaryPath() != "from" || fm.SecondaryPath() != "to" {
		t.Errorf("file_move paths wrong: primary=%q secondary=%q", fm.PrimaryPath(), fm.SecondaryPath())
	}
	ir := Operation{ImportRewrite: &ImportRewrite{OldPath: "old", NewPath: "new"}}
	if ir.PrimaryPath() != "old" || ir.SecondaryPath() != "new" {
		t.Errorf("import_rewrite paths wrong: primary=%q secondary=%q", ir.PrimaryPath(), ir.SecondaryPath())
	}
	empty := Operation{}
	if empty.PrimaryPath() != "" || empty.SecondaryPath() != "" {
		t.Errorf("empty paths must return empty strings")
	}
}

func TestRewriteErrorString(t *testing.T) {
	e := RewriteError{Kind: RewriteErrorInvalidInput, Path: "/abs", Message: "boom"}
	if e.Error() == "" {
		t.Error("expected non-empty error string")
	}
}
