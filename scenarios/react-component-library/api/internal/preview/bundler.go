package preview

import (
	"context"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

// Esbuilder is the production Bundler. It uses esbuild's Transform
// (not Build) API: each component is one TSX file with bare imports,
// and the importmap on the harness HTML resolves `react` /
// `react-dom/client` client-side. Transform is cheaper than Build and
// matches the "one file in, one file out" reality of slice 3.
type Esbuilder struct{}

// NewEsbuilder constructs the production Bundler.
func NewEsbuilder() *Esbuilder { return &Esbuilder{} }

// BuildBundle transforms TSX → ES module text. sourcePath is forwarded
// to esbuild as Sourcefile so its diagnostic messages reference the
// human-recognisable name instead of "<stdin>".
func (Esbuilder) BuildBundle(_ context.Context, tsx string, sourcePath string) (string, []string, error) {
	result := esbuild.Transform(tsx, esbuild.TransformOptions{
		Loader:     esbuild.LoaderTSX,
		Format:     esbuild.FormatESModule,
		Target:     esbuild.ES2020,
		JSX:        esbuild.JSXAutomatic,
		Sourcefile: sourcePath,
		LogLevel:   esbuild.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		msgs := make([]string, 0, len(result.Errors))
		for _, m := range result.Errors {
			msgs = append(msgs, m.Text)
		}
		return "", nil, ErrBundle{SourcePath: sourcePath, Messages: msgs}
	}
	warns := make([]string, 0, len(result.Warnings))
	for _, m := range result.Warnings {
		warns = append(warns, m.Text)
	}
	return string(result.Code), warns, nil
}
