package cliapp

import (
	"strings"
	"testing"
)

func TestArgSchemaValidate(t *testing.T) {
	t.Run("empty schema is valid", func(t *testing.T) {
		if err := (ArgSchema{}).Validate(); err != nil {
			t.Fatalf("empty schema: %v", err)
		}
	})

	t.Run("simple flags+positionals", func(t *testing.T) {
		s := ArgSchema{
			Positionals: []Positional{{Name: "id", Required: true}},
			Flags: []Flag{
				{Name: "title", Required: true},
				{Name: "verbose", Aliases: []string{"v"}, Bool: true},
			},
		}
		if err := s.Validate(); err != nil {
			t.Fatalf("valid schema rejected: %v", err)
		}
	})

	t.Run("repeated must be last", func(t *testing.T) {
		s := ArgSchema{Positionals: []Positional{
			{Name: "first", Repeated: true},
			{Name: "second"},
		}}
		err := s.Validate()
		if err == nil || !strings.Contains(err.Error(), "Repeated") {
			t.Fatalf("expected Repeated-not-last error, got: %v", err)
		}
	})

	t.Run("required after optional rejected", func(t *testing.T) {
		s := ArgSchema{Positionals: []Positional{
			{Name: "a"},
			{Name: "b", Required: true},
		}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected error: required positional after optional")
		}
	})

	t.Run("duplicate flag name", func(t *testing.T) {
		s := ArgSchema{Flags: []Flag{
			{Name: "x"},
			{Name: "x"},
		}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected duplicate-name error")
		}
	})

	t.Run("flag with leading dash rejected", func(t *testing.T) {
		s := ArgSchema{Flags: []Flag{{Name: "--title"}}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected leading-dash error")
		}
	})

	t.Run("alias collision with positional", func(t *testing.T) {
		s := ArgSchema{
			Positionals: []Positional{{Name: "x"}},
			Flags:       []Flag{{Name: "y", Aliases: []string{"x"}}},
		}
		if err := s.Validate(); err == nil {
			t.Fatal("expected alias-collision error")
		}
	})

	t.Run("bool with default rejected", func(t *testing.T) {
		s := ArgSchema{Flags: []Flag{{Name: "verbose", Bool: true, Default: "true"}}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected bool+default error")
		}
	})

	t.Run("values declarations", func(t *testing.T) {
		cases := []struct {
			name    string
			flag    Flag
			wantErr string // "" = valid
		}{
			{
				name: "values with aliases and default",
				flag: Flag{
					Name:         "complexity",
					Values:       []string{"minor", "moderate", "major", "architectural"},
					ValueAliases: map[string]string{"low": "minor", "medium": "moderate", "high": "major"},
					Default:      "moderate",
				},
			},
			{
				name: "default may be an alias",
				flag: Flag{
					Name:         "complexity",
					Values:       []string{"minor", "moderate"},
					ValueAliases: map[string]string{"low": "minor"},
					Default:      "low",
				},
			},
			{
				name:    "values on bool flag rejected",
				flag:    Flag{Name: "verbose", Bool: true, Values: []string{"yes", "no"}},
				wantErr: "Values requires a valued flag",
			},
			{
				name:    "empty value rejected",
				flag:    Flag{Name: "kind", Values: []string{"a", " "}},
				wantErr: "is empty",
			},
			{
				name:    "duplicate value rejected",
				flag:    Flag{Name: "kind", Values: []string{"a", "a"}},
				wantErr: "duplicate",
			},
			{
				name:    "alias to undeclared value rejected",
				flag:    Flag{Name: "kind", Values: []string{"a"}, ValueAliases: map[string]string{"b": "c"}},
				wantErr: "not a declared value",
			},
			{
				name:    "alias shadowing a value rejected",
				flag:    Flag{Name: "kind", Values: []string{"a", "b"}, ValueAliases: map[string]string{"a": "b"}},
				wantErr: "duplicates a declared value",
			},
			{
				name:    "aliases without values rejected",
				flag:    Flag{Name: "kind", ValueAliases: map[string]string{"a": "b"}},
				wantErr: "ValueAliases without Values",
			},
			{
				name:    "default outside vocabulary rejected",
				flag:    Flag{Name: "kind", Values: []string{"a", "b"}, Default: "c"},
				wantErr: "neither a declared value nor an alias",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := (ArgSchema{Flags: []Flag{tc.flag}}).Validate()
				if tc.wantErr == "" {
					if err != nil {
						t.Fatalf("valid flag rejected: %v", err)
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
				}
			})
		}
	})
}

func TestArgSchemaFlagByName(t *testing.T) {
	s := ArgSchema{Flags: []Flag{
		{Name: "title"},
		{Name: "verbose", Aliases: []string{"v", "verb"}},
	}}
	for _, name := range []string{"title", "verbose", "v", "verb"} {
		if _, ok := s.flagByName(name); !ok {
			t.Errorf("flagByName(%q) not found", name)
		}
	}
	if _, ok := s.flagByName("missing"); ok {
		t.Error("flagByName(missing) should not match")
	}
}
