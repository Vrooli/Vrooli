package main

import (
	"strings"
	"testing"
)

func TestCheckMakefileLifecycle_Valid(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
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
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
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

func TestCheckMakefileLifecycle_MessageMismatch(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)
`
	violations, err := CheckMakefileLifecycle(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if !strings.Contains(violations[0].Message, "start target must echo") {
		t.Errorf("unexpected message: %s", violations[0].Message)
	}
}

func TestCheckMakefileLifecycle_Multiline(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start \
		"$(SCENARIO_NAME)"

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop "$(SCENARIO_NAME)"

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test "$(SCENARIO_NAME)"

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs "$(SCENARIO_NAME)" \
		--tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
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
	@+echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@+vrooli scenario start $(SCENARIO_NAME)

stop:
	@+echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@+vrooli scenario stop $(SCENARIO_NAME)

test:
	@+echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@+vrooli scenario test $(SCENARIO_NAME)

logs:
	@+echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@+vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@+echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
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
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail=50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
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
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME) --detach

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
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
// is accepted as equivalent to $(SCENARIO_NAME) in both commands and echo messages.
func TestCheckMakefileLifecycle_CurlyBraceVarSyntax(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "${BLUE}🚀 Starting ${SCENARIO_NAME} scenario...${RESET}"
	@vrooli scenario start ${SCENARIO_NAME}

stop:
	@echo "${YELLOW}⏹️  Stopping ${SCENARIO_NAME} scenario...${RESET}"
	@vrooli scenario stop ${SCENARIO_NAME}

test:
	@echo "${BLUE}🧪 Testing ${SCENARIO_NAME} scenario...${RESET}"
	@vrooli scenario test ${SCENARIO_NAME}

logs:
	@echo "${BLUE}📜 Logs for ${SCENARIO_NAME}:${RESET}"
	@vrooli scenario logs ${SCENARIO_NAME} --tail 50

status:
	@echo "${BLUE}📊 Status of ${SCENARIO_NAME}:${RESET}"
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

// TestCheckMakefileLifecycle_MixedVarSyntax verifies that mixing $() and ${}
// in the same Makefile is accepted.
func TestCheckMakefileLifecycle_MixedVarSyntax(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting ${SCENARIO_NAME} scenario...$(RESET)"
	@vrooli scenario start ${SCENARIO_NAME}

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
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

func TestNormalizeVarSyntax(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"$(SCENARIO_NAME)", "$(SCENARIO_NAME)"},
		{"${SCENARIO_NAME}", "$(SCENARIO_NAME)"},
		{"${BLUE}text${RESET}", "$(BLUE)text$(RESET)"},
		{"no vars here", "no vars here"},
		{"$(BLUE)${SCENARIO_NAME}$(RESET)", "$(BLUE)$(SCENARIO_NAME)$(RESET)"},
	}
	for _, tt := range tests {
		got := normalizeVarSyntax(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeVarSyntax(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCheckMakefileLifecycle_LogsExtraFlags(t *testing.T) {
	content := `SCENARIO_NAME := $(notdir $(CURDIR))

start:
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test:
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50 --follow

status:
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
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
