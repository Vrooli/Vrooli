package main

import (
	"strings"
	"testing"
)

// TestCheckMakefileLifecycle_ThinWrapper verifies that a Makefile whose lifecycle
// targets contain ONLY the canonical `vrooli scenario <verb>` command (no echoes,
// no banners) passes the rule. Banners are owned by the CLI, not the Makefile.
func TestCheckMakefileLifecycle_ThinWrapper(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@vrooli scenario stop $(SCENARIO_NAME)

restart:
	@vrooli scenario restart $(SCENARIO_NAME)

test:
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@vrooli scenario status $(SCENARIO_NAME)
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

// TestCheckMakefileLifecycle_EchoesAllowed verifies that scenarios which still
// have legacy @echo lines before the canonical command are NOT flagged. The rule
// no longer cares about echoes — it only checks for the canonical command.
func TestCheckMakefileLifecycle_EchoesAllowed(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "anything you want here"
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@vrooli scenario stop $(SCENARIO_NAME)

restart:
	@vrooli scenario restart $(SCENARIO_NAME)

test:
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@vrooli scenario status $(SCENARIO_NAME)
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

func TestCheckMakefileLifecycle_LegacyRun(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@vrooli scenario run $(SCENARIO_NAME)

stop:
	@vrooli scenario stop $(SCENARIO_NAME)

restart:
	@vrooli scenario restart $(SCENARIO_NAME)

test:
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@vrooli scenario status $(SCENARIO_NAME)
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if !strings.Contains(violations[0].Message, "execute 'vrooli scenario start") {
		t.Errorf("expected message about 'execute vrooli scenario start', got: %s", violations[0].Message)
	}
}

func TestCheckMakefileLifecycle_Multiline(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@vrooli scenario start \
		"$(SCENARIO_NAME)"

stop:
	@vrooli scenario stop "$(SCENARIO_NAME)"

restart:
	@vrooli scenario restart "$(SCENARIO_NAME)"

test:
	@vrooli scenario test "$(SCENARIO_NAME)"

logs:
	@vrooli scenario logs "$(SCENARIO_NAME)" \
		--tail 50

status:
	@vrooli scenario status "$(SCENARIO_NAME)"
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

func TestCheckMakefileLifecycle_RecursivePrefix(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@+vrooli scenario start $(SCENARIO_NAME)

stop:
	@+vrooli scenario stop $(SCENARIO_NAME)

restart:
	@+vrooli scenario restart $(SCENARIO_NAME)

test:
	@+vrooli scenario test $(SCENARIO_NAME)

logs:
	@+vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@+vrooli scenario status $(SCENARIO_NAME)
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

func TestCheckMakefileLifecycle_LogsTailEquals(t *testing.T) {
	// Both "--tail 50" and "--tail=50" are valid Go/shell flag formats.
	// The rule should accept either form.
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@vrooli scenario stop $(SCENARIO_NAME)

restart:
	@vrooli scenario restart $(SCENARIO_NAME)

test:
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@vrooli scenario logs $(SCENARIO_NAME) --tail=50

status:
	@vrooli scenario status $(SCENARIO_NAME)
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

func TestCheckMakefileLifecycle_StartExtraFlags(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@vrooli scenario start $(SCENARIO_NAME) --detach

stop:
	@vrooli scenario stop $(SCENARIO_NAME)

restart:
	@vrooli scenario restart $(SCENARIO_NAME)

test:
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@vrooli scenario status $(SCENARIO_NAME)
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Extra flags after the 4 required tokens should be accepted.
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

// TestCheckMakefileLifecycle_CurlyBraceVarSyntax verifies that ${SCENARIO_NAME}
// is accepted as equivalent to $(SCENARIO_NAME) in the canonical command.
func TestCheckMakefileLifecycle_CurlyBraceVarSyntax(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@vrooli scenario start ${SCENARIO_NAME}

stop:
	@vrooli scenario stop ${SCENARIO_NAME}

restart:
	@vrooli scenario restart ${SCENARIO_NAME}

test:
	@vrooli scenario test ${SCENARIO_NAME}

logs:
	@vrooli scenario logs ${SCENARIO_NAME} --tail 50

status:
	@vrooli scenario status ${SCENARIO_NAME}
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

func TestCheckMakefileLifecycle_LogsExtraFlags(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@vrooli scenario stop $(SCENARIO_NAME)

restart:
	@vrooli scenario restart $(SCENARIO_NAME)

test:
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50 --follow

status:
	@vrooli scenario status $(SCENARIO_NAME)
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Extra flags after --tail 50 should be accepted (>= 6 tokens).
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

func TestCheckMakefileLifecycle_MissingTarget(t *testing.T) {
	// A scenario that omits a lifecycle target entirely should be flagged.
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@vrooli scenario stop $(SCENARIO_NAME)

restart:
	@vrooli scenario restart $(SCENARIO_NAME)

logs:
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@vrooli scenario status $(SCENARIO_NAME)
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for missing test target, got %d", len(violations))
	}
	if !strings.Contains(violations[0].Message, "test target missing") {
		t.Errorf("expected 'test target missing' violation, got: %s", violations[0].Message)
	}
}
