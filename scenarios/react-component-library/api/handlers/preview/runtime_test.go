package preview

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackageRuntimeEntryResolvesPackageExports(t *testing.T) {
	tests := []struct {
		name        string
		moduleName  string
		packageJSON string
		want        string
	}{
		{
			name:       "exports only import entry",
			moduleName: "tailwind-merge",
			packageJSON: `{
  "exports": {
    ".": {
      "types": "./dist/types.d.ts",
      "require": "./dist/bundle-cjs.js",
      "import": "./dist/bundle-mjs.mjs",
      "default": "./dist/bundle-mjs.mjs"
    }
  },
  "main": "./dist/bundle-cjs.js"
}`,
			want: filepath.Join("tailwind-merge", "dist", "bundle-mjs.mjs"),
		},
		{
			name:       "browser condition wins",
			moduleName: "browser-first",
			packageJSON: `{
  "exports": {
    ".": {
      "browser": "./dist/browser.js",
      "import": "./dist/import.js"
    }
  }
}`,
			want: filepath.Join("browser-first", "dist", "browser.js"),
		},
		{
			name:       "string root export",
			moduleName: "string-export",
			packageJSON: `{
  "exports": {
    ".": "./dist/index.mjs"
  },
  "module": "./dist/module.js"
}`,
			want: filepath.Join("string-export", "dist", "index.mjs"),
		},
		{
			name:       "nested conditional export",
			moduleName: "nested-export",
			packageJSON: `{
  "exports": {
    ".": {
      "browser": {
        "import": "./dist/browser-import.mjs",
        "default": "./dist/browser-default.js"
      },
      "import": "./dist/import.mjs"
    }
  }
}`,
			want: filepath.Join("nested-export", "dist", "browser-import.mjs"),
		},
		{
			name:       "module field fallback",
			moduleName: "clsx",
			packageJSON: `{
  "module": "./dist/clsx.mjs",
  "main": "./dist/clsx.js"
}`,
			want: filepath.Join("clsx", "dist", "clsx.mjs"),
		},
		{
			name:       "main field fallback",
			moduleName: "main-only",
			packageJSON: `{
  "main": "./dist/index.cjs"
}`,
			want: filepath.Join("main-only", "dist", "index.cjs"),
		},
		{
			name:       "malformed exports falls back",
			moduleName: "bad-exports",
			packageJSON: `{
  "exports": {
    ".": {
      "import": 42
    }
  },
  "module": "./dist/fallback.mjs"
}`,
			want: filepath.Join("bad-exports", "dist", "fallback.mjs"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTempNodeModulesPackage(t, tt.moduleName, tt.packageJSON)

			got, ok := packageRuntimeEntry(tt.moduleName, "index.js")

			require.True(t, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPackageRuntimeEntryRejectsUnsafeExportEntry(t *testing.T) {
	withTempNodeModulesPackage(t, "unsafe-export", `{
  "exports": {
    ".": {
      "import": "../escape.mjs"
    }
  }
}`)

	_, ok := packageRuntimeEntry("unsafe-export", "index.js")

	require.False(t, ok)
}

func TestPackageRuntimeEntryPreservesExplicitRuntimeSubpath(t *testing.T) {
	got, ok := packageRuntimeEntry("tailwind-merge", "dist/bundle-mjs.mjs")

	require.True(t, ok)
	require.Equal(t, filepath.Join("tailwind-merge", "dist", "bundle-mjs.mjs"), got)
}

func withTempNodeModulesPackage(t *testing.T, moduleName, packageJSON string) {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	dir := filepath.Join(root, "ui", "node_modules", filepath.FromSlash(moduleName))
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0o600))
}
