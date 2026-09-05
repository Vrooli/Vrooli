package rules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// evalScenarioPrivilegeBoundary is deliberately source-semantic. A textual
// grep misses exec wrappers and produces false positives for documentation;
// AST matching lets the rule identify the actual command invocation and its
// source location.
func evalScenarioPrivilegeBoundary(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleScenarioPrivilegeBoundary)
	root := ctx.Surface.RootPath
	if root == "" {
		root = ctx.Inventory.RootPath
	}
	if root == "" {
		return nil
	}
	grants := privilegeGrants(repoRoot(root))
	var findings []Finding
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "node_modules", "dist", "build", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || privilegeSurfaceExempt(path) {
			return nil
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return nil
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			argv, matched := privilegeCall(call)
			if !matched || coveredPrivilege(argv, grants) {
				return true
			}
			position := fset.Position(call.Pos())
			findings = append(findings, ruleFinding(ctx, rule, path,
				"runtime privilege escalation is outside the setup-provisioned boundary",
				"line "+itoa(position.Line)+": "+strings.Join(argv, " "),
				"no direct sudo/pkexec/doas/runas invocation in scenario runtime code",
				"line "+itoa(position.Line)+": "+strings.Join(argv, " ")+"; use a setup safeguard or privilege-broker action"))
			return true
		})
		return nil
	})
	return findings
}

type privilegeGrant struct {
	command string
	action  string
	unit    string
}

func privilegeCall(call *ast.CallExpr) ([]string, bool) {
	if call == nil {
		return nil, false
	}
	name := ""
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		name = fun.Sel.Name
	case *ast.Ident:
		name = fun.Name
	}
	allowedMethod := map[string]bool{
		"Command": true, "CommandContext": true, "Output": true,
		"CombinedOutput": true, "Run": true, "Start": true,
		"commandOutput": true, "commandRun": true, "commandOutputInput": true,
	}
	if !allowedMethod[name] {
		return nil, false
	}
	var argv []string
	for _, arg := range call.Args {
		literal, ok := arg.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			continue
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			value = strings.Trim(literal.Value, "\"")
		}
		argv = append(argv, value)
	}
	if len(argv) > 0 && (argv[0] == "sudo" || argv[0] == "pkexec" || argv[0] == "doas" || argv[0] == "runas") {
		return argv, true
	}
	joined := strings.ToLower(strings.Join(argv, " "))
	if (strings.Contains(joined, "start-process") && strings.Contains(joined, "-verb") && strings.Contains(joined, "runas")) ||
		(strings.Contains(joined, "osascript") && strings.Contains(joined, "administrator privileges")) ||
		strings.Contains(joined, "/etc/sudoers") {
		return argv, true
	}
	return nil, false
}

func privilegeSurfaceExempt(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	for _, prefix := range []string{"/internal/setup/", "/internal/hostreqkit/", "/internal/safeguards/", "/internal/privilegebroker/"} {
		if strings.Contains(clean, prefix) {
			return true
		}
	}
	return false
}

func repoRoot(path string) string {
	path, _ = filepath.Abs(path)
	if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
		path = filepath.Dir(path)
	}
	for {
		if exists(filepath.Join(path, ".git")) || exists(filepath.Join(path, "AGENTS.md")) {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			return ""
		}
		path = parent
	}
}

func privilegeGrants(root string) []privilegeGrant {
	if root == "" {
		return nil
	}
	var grants []privilegeGrant
	pattern := regexp.MustCompile(`\b(start|restart|stop|reset-failed)\s+([[:alnum:]_.@-]+)\b`)
	_ = filepath.WalkDir(filepath.Join(root, "internal", "safeguards"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() || entry.Name() != "handler.go" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		command := ""
		if match := regexp.MustCompile(`systemctlPath\s*=\s*"([^"]+)"`).FindSubmatch(raw); len(match) == 2 {
			command = string(match[1])
		}
		for _, match := range pattern.FindAllSubmatch(raw, -1) {
			if command == "" {
				command = "systemctl"
			}
			grants = append(grants, privilegeGrant{command: filepath.Base(command), action: string(match[1]), unit: string(match[2])})
		}
		return nil
	})
	sort.Slice(grants, func(i, j int) bool { return grants[i].action+grants[i].unit < grants[j].action+grants[j].unit })
	return grants
}

func coveredPrivilege(argv []string, grants []privilegeGrant) bool {
	if len(argv) < 4 || argv[0] != "sudo" {
		return false
	}
	commandIndex := 1
	if argv[commandIndex] == "-n" {
		commandIndex++
	}
	if len(argv) <= commandIndex+2 {
		return false
	}
	command := filepath.Base(argv[commandIndex])
	for _, grant := range grants {
		if command == grant.command && argv[commandIndex+1] == grant.action && argv[commandIndex+2] == grant.unit {
			return true
		}
	}
	return false
}

func itoa(value int) string {
	if value < 0 {
		return "?"
	}
	return strconv.Itoa(value)
}
