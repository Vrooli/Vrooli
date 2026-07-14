package themes_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/themes"
	thmocks "react-component-library/internal/themes/mocks"
)

type fakeDesigns struct {
	bytesByScenario map[string][]byte
	err             error
}

func (f *fakeDesigns) Read(_ context.Context, scenario string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	b, ok := f.bytesByScenario[scenario]
	if !ok {
		return nil, errors.New("missing")
	}
	return b, nil
}

func TestEnsureBuiltinsSeeded_PopulatesWhenEmpty(t *testing.T) {
	repo := thmocks.NewFakeRepository()
	svc := themes.NewService(repo, nil)
	require.NoError(t, svc.EnsureBuiltinsSeeded(context.Background()))
	list, err := svc.ListBuiltins(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 3)
	ids := map[string]bool{}
	for _, t := range list {
		ids[t.ID] = true
	}
	require.True(t, ids["light"])
	require.True(t, ids["dark"])
	require.True(t, ids["high-contrast"])
}

func TestBuiltinThemes_EmitOnlyConsumedTailwindContractTokens(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "ui", "tailwind.theme.json"))
	require.NoError(t, err)
	var contract struct {
		Colors       map[string]string `json:"colors"`
		BorderRadius map[string]string `json:"borderRadius"`
	}
	require.NoError(t, json.Unmarshal(raw, &contract))
	consumed := map[string]bool{}
	for _, value := range contract.Colors {
		var token string
		_, _ = fmt.Sscanf(value, "var(%s)", &token)
		consumed[strings.TrimSuffix(token, ")")] = true
	}
	for _, value := range contract.BorderRadius {
		var token string
		_, _ = fmt.Sscanf(value, "var(%s)", &token)
		consumed[strings.TrimSuffix(token, ")")] = true
	}
	repo := thmocks.NewFakeRepository()
	svc := themes.NewService(repo, nil)
	require.NoError(t, svc.EnsureBuiltinsSeeded(context.Background()))
	builtins, err := svc.ListBuiltins(context.Background())
	require.NoError(t, err)
	for _, theme := range builtins {
		for token := range theme.Tokens {
			require.Truef(t, consumed[token], "%s emits unused token %s", theme.ID, token)
		}
	}
}

func TestEnsureBuiltinsSeeded_NoopWhenNonEmpty(t *testing.T) {
	repo := thmocks.NewFakeRepository()
	svc := themes.NewService(repo, nil)
	require.NoError(t, svc.EnsureBuiltinsSeeded(context.Background()))
	listFirst, _ := svc.ListBuiltins(context.Background())
	require.NoError(t, svc.EnsureBuiltinsSeeded(context.Background()))
	listSecond, _ := svc.ListBuiltins(context.Background())
	require.Equal(t, len(listFirst), len(listSecond))
}

func TestGetBuiltin_NotFound(t *testing.T) {
	repo := thmocks.NewFakeRepository()
	svc := themes.NewService(repo, nil)
	_, err := svc.GetBuiltin(context.Background(), "nope")
	require.Error(t, err)
	var sentinel themes.ErrThemeNotFound
	require.ErrorAs(t, err, &sentinel)
}

func TestResolveFromScenario_OK(t *testing.T) {
	repo := thmocks.NewFakeRepository()
	designs := &fakeDesigns{bytesByScenario: map[string][]byte{
		"flow-verifier": []byte(flowVerifierDesignMD),
	}}
	svc := themes.NewService(repo, designs)
	theme, err := svc.ResolveFromScenario(context.Background(), "flow-verifier")
	require.NoError(t, err)
	require.Equal(t, "scenario:flow-verifier", theme.Source)
	require.NotEmpty(t, theme.Tokens)
}

func TestResolveFromScenario_Missing(t *testing.T) {
	repo := thmocks.NewFakeRepository()
	designs := &fakeDesigns{err: errors.New("ENOENT")}
	svc := themes.NewService(repo, designs)
	_, err := svc.ResolveFromScenario(context.Background(), "scn")
	require.Error(t, err)
	var sentinel themes.ErrScenarioDesignMDMissing
	require.ErrorAs(t, err, &sentinel)
}
