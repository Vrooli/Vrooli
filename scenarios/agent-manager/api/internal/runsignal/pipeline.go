package runsignal

import (
	"path/filepath"
	"strings"

	"agent-manager/internal/domain"
)

// Pipeline is one command as the shell runs it: its words (program first,
// wrappers stripped, the program path-stripped) and the commands its output is
// piped through, each program first. Loop is set when the command runs inside a
// while/until loop, which re-runs it until a condition holds.
type Pipeline struct {
	Words   []string
	Filters [][]string
	Loop    bool
}

// CallPipelines reads the commands a tool call ran from its own input: the
// shell text or argv of a Bash or exec_command call, and each literal
// exec_command of Codex exec JavaScript. It uses the same lexer as the facts,
// so a fact's command path names one of these pipelines.
func CallPipelines(call *domain.ToolCallEventData) []Pipeline {
	if call == nil {
		return nil
	}
	if src, ok := jsExecInput(call); ok {
		commands, _, _ := jsExecCommands(src)
		var out []Pipeline
		for _, command := range commands {
			if len(command.Args) > 0 {
				out = append(out, ArgvPipelines(command.Args)...)
			} else {
				out = append(out, shellPipelines(command.Script, false)...)
			}
		}
		return out
	}
	extraction := genericCommandInput(call)
	if extraction.LiteralArgs {
		return ArgvPipelines(extraction.Args)
	}
	return shellPipelines(extraction.Command, false)
}

// ArgvPipelines reads a literal argv: words are never reparsed as syntax,
// except the script of a `bash -lc` wrapper.
func ArgvPipelines(argv []string) []Pipeline {
	tokens := stripWrappers(argv)
	if len(tokens) == 0 {
		return nil
	}
	if _, script, wrapped := shellScript(tokens); wrapped {
		return shellPipelines(script, false)
	}
	return []Pipeline{{Words: append([]string{filepath.Base(tokens[0])}, tokens[1:]...)}}
}

// shellPipelines groups the lexed segments of shell text into pipelines. A
// `bash -lc '<script>'` segment contributes the pipelines of its script.
func shellPipelines(text string, loop bool) []Pipeline {
	segments, ok := lexShell(text)
	if !ok {
		return nil
	}
	var out []Pipeline
	var current *Pipeline
	piped := false
	for _, segment := range segments {
		tokens := segment.Tokens
		for len(tokens) > 0 && controlPrefixes[tokens[0]] {
			if tokens[0] == "while" || tokens[0] == "until" {
				loop = true
			}
			tokens = tokens[1:]
		}
		if len(tokens) > 0 && tokens[0] == "done" {
			loop = false
		}
		words := []string(nil)
		if len(tokens) > 0 && !controlOnly[tokens[0]] {
			words = stripWrappers(tokens)
		}
		switch {
		case len(words) == 0:
		case piped && current != nil:
			current.Filters = append(current.Filters, append([]string{filepath.Base(words[0])}, words[1:]...))
		default:
			if _, script, wrapped := shellScript(words); wrapped {
				out = append(out, shellPipelines(script, loop)...)
				current = nil
				break
			}
			out = append(out, Pipeline{Words: append([]string{filepath.Base(words[0])}, words[1:]...), Loop: loop})
			current = &out[len(out)-1]
		}
		piped = segment.Op == "|"
		if !piped {
			current = nil
		}
	}
	return out
}

// PipelineFor finds the pipeline a command fact was made from: the n-th
// pipeline (counting from 0) whose program and subcommand words begin the
// fact's command path.
func PipelineFor(pipelines []Pipeline, commandPath string, n int) (Pipeline, bool) {
	words := strings.Fields(commandPath)
	if len(words) == 0 {
		return Pipeline{}, false
	}
	for _, pipeline := range pipelines {
		if len(pipeline.Words) == 0 || pipeline.Words[0] != words[0] || !containsInOrder(pipeline.Words[1:], words[1:]) {
			continue
		}
		if n == 0 {
			return pipeline, true
		}
		n--
	}
	return Pipeline{}, false
}

// containsInOrder reports whether want appears in words in order, skipping
// flags and their values between them.
func containsInOrder(words, want []string) bool {
	i := 0
	for _, word := range words {
		if i < len(want) && word == want[i] {
			i++
		}
	}
	return i == len(want)
}
