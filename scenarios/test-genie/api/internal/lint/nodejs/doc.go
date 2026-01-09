// Package nodejs provides TypeScript/JavaScript linting using tsc and eslint.
//
// Tools used:
//   - tsc --noEmit: Type checking (if tsconfig.json exists)
//   - eslint: Linting (if eslint config exists)
//
// Type errors from tsc are treated as errors that fail the phase.
// ESLint issues are treated as warnings that don't fail the phase.
package nodejs
