// Package packages detects required package managers for a scenario.
//
// Detection is based on lockfile presence in the scenario directory:
//   - pnpm: pnpm-lock.yaml in root or ui/
//   - npm: package-lock.json in root or ui/
//   - yarn: yarn.lock in root or ui/
//
// If a Node.js workspace is detected but no lockfile exists, pnpm is assumed
// as the default package manager.
//
// The package provides an interface for testing seams and supports dependency
// injection for isolated unit tests.
package packages
