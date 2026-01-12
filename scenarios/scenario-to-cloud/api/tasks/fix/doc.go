// Package fix implements the iterative fix task handler.
//
// Fix tasks apply fixes to deployment failures through an iterative loop:
//  1. Diagnose: Assess current state and identify issues
//  2. Change: Apply fixes based on permissions (immediate/permanent/prevention)
//  3. Deploy: Trigger redeployment if needed
//  4. Verify: Check if the fix resolved the issue
//  5. Repeat: Continue until success, max iterations, or agent gives up
//
// Permission types control what fixes are allowed:
//   - immediate: VPS hotfixes (restart services, clear disk, etc.)
//   - permanent: Code/configuration changes in the codebase
//   - prevention: Monitoring, alerts, or pipeline improvements
//
// Focus options control which components to examine:
//   - harness: scenario-to-cloud deployment infrastructure
//   - subject: the target scenario being deployed
//
// The default maximum iterations is 5, configurable per-request.
package fix
