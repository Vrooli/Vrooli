// Package shell provides a syntax linter for shell scripts.
//
// The linter validates shell script syntax using `bash -n`, which performs
// a syntax check without executing the script.
//
// Detection:
//   - Looks for CLI binary in cli/ directory (scenario-name or fallback patterns)
//   - Verifies the file is executable
//   - Verifies `bash` command is available
//
// Execution:
//   - Runs `bash -n <script>` on each shell entrypoint
//   - Reports syntax errors without executing the scripts
package shell
