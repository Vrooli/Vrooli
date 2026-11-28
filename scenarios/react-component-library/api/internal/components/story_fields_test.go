package components

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeriveStoryFieldsExtractsOwnedProps(t *testing.T) {
	source := `
export interface ExampleProps extends React.HTMLAttributes<HTMLDivElement> {
  /** The display label. @default Ready */
  label?: string;
  /** The visual mode. */
  mode: "compact" | "comfortable";
  disabled?: boolean;
  count: number;
  children: React.ReactNode;
  items: string[];
  onChange?: (value: string) => void;
}
`

	got := DeriveStoryFields(source)
	require.Equal(t, []StoryField{
		{Path: "children", Kind: StoryFieldText, Required: true},
		{Path: "count", Kind: StoryFieldNumber, Required: true},
		{Path: "disabled", Kind: StoryFieldBoolean},
		{Path: "items", Kind: StoryFieldArray, Required: true},
		{Path: "label", Label: "The display label", Kind: StoryFieldText, Default: []byte(`"Ready"`)},
		{Path: "mode", Label: "The visual mode", Kind: StoryFieldEnum, Required: true, Options: []json.RawMessage{json.RawMessage(`"compact"`), json.RawMessage(`"comfortable"`)}},
	}, got)
}

func TestDeriveStoryFieldsOmitsOpaqueAndInheritedProps(t *testing.T) {
	source := `export type ExampleProps = { className?: string; value: unknown; title?: string; }`
	got := DeriveStoryFields(source)
	require.Equal(t, []StoryField{{Path: "title", Kind: StoryFieldText}}, got)
}
