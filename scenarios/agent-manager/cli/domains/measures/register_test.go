package measures

import "testing"

func TestRegisterDeclaresMeasureCommands(t *testing.T) {
	group := Register()
	if len(group.Subcommands) == 0 {
		t.Fatal("measure command group must expose its declared measures")
	}
}
