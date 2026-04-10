// Package investigate implements the investigation task handler.
//
// Investigation tasks analyze deployment failures to identify root causes
// and recommend fixes. The agent examines logs, checks service status,
// and produces a structured report with findings and recommendations.
//
// Effort levels control investigation depth:
//   - checks: Quick health checks and basic diagnostics
//   - logs: Log analysis and standard diagnostic procedures
//   - trace: Full SSH tracing and deep analysis
//
// Focus options control which components are examined:
//   - harness: scenario-to-cloud deployment infrastructure
//   - subject: the target scenario being deployed
package investigate
