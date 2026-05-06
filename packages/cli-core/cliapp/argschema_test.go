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
