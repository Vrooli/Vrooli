package rootclitest

import (
	"errors"
	"testing"
)

func TestValidateHelpWithNoArgs(t *testing.T) {
	tests := []struct {
		name    string
		run     func() error
		output  string
		usage   string
		wantErr bool
	}{
		{name: "valid", run: func() error { return nil }, output: "Usage: vrooli resource", usage: "vrooli resource"},
		{name: "handler error", run: func() error { return errors.New("boom") }, output: "Usage: vrooli resource", usage: "vrooli resource", wantErr: true},
		{name: "missing usage mutant", run: func() error { return nil }, output: "Usage: vrooli resource", usage: "vrooli package", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateHelpWithNoArgs(test.run, func() string { return test.output }, test.usage)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateHelpWithNoArgs() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
