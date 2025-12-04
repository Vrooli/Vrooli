// Package nodejs provides a unit test runner for Node.js projects.
//
// The runner detects Node.js projects by checking for package.json,
// determines the appropriate package manager (pnpm, yarn, or npm),
// and executes the test script defined in package.json.
//
// Detection:
//   - Checks for ui/package.json or package.json in scenario root
//   - Verifies `node` command is available
//   - Verifies package manager is available
//
// Package Manager Detection (in priority order):
//  1. packageManager field in package.json (e.g., "pnpm@8.0.0")
//  2. pnpm-lock.yaml presence -> pnpm
//  3. yarn.lock presence -> yarn
//  4. Default to npm
//
// Execution:
//   - Installs dependencies if node_modules is missing
//   - Runs the package manager's test command
//   - Extracts coverage from coverage-summary.json if available
package nodejs
