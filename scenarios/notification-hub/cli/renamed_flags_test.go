package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenamedFlagsFailAsUnknownOptionsAndNameTheNewFlag(t *testing.T) {
	cases := map[string]struct {
		args []string
		want string
	}{
		"ask expiry": {[]string{"conversations", "ask", "--question", "q", "--deadline", "6h"}, "--deadline was renamed to --expires"},
		"ask wait":   {[]string{"conversations", "ask", "--question", "q", "--and-wait", "--wait-for=10m"}, "--wait-for was renamed to --timeout"},
		"wait":       {[]string{"conversations", "wait", "--deadline", "30m", "abc"}, "--deadline was renamed to --timeout"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			app, err := NewApp()
			require.NoError(t, err)
			err = app.Run(tc.args)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unknown option")
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestOtherUnknownFlagsAreLeftAlone(t *testing.T) {
	require.Nil(t, explainRenamedFlag([]string{"conversations", "get", "--deadline", "1h"}, nil))
	err := explainRenamedFlag([]string{"conversations", "get", "--deadline", "1h"}, errString("unknown option: --deadline"))
	require.EqualError(t, err, "unknown option: --deadline")
}

type errString string

func (e errString) Error() string { return string(e) }
