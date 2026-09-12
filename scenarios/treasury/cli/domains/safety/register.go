// Package safety exposes Treasury's durable financial safety constraints.
package safety

import "github.com/vrooli/cli-core/cliapp"

// Invariants returns the constraints that implementation phases must preserve.
// Keeping this list executable gives operators a stable local reference even
// when Treasury or one of its runtime dependencies is unavailable.
func Invariants() []string {
	return []string{
		"AgentSpend exposes no policy-mutating method.",
		"Unverifiable agent identity fails closed.",
		"Only operator-owned funds are representable.",
		"Headroom is computed from records and is never stored as a balance.",
		"Settlement retries never move value twice for one idempotency key.",
		"Evidence is append-only and retained for every spend attempt.",
	}
}

// Register returns the local safety inspection group. It deliberately needs no
// API: operators must be able to inspect these invariants during an outage.
func Register() cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "safety",
		Description: "Inspect Treasury financial safety invariants",
		Subcommands: []cliapp.Command{
			{
				Name:        "invariants",
				Description: "Show the constraints every implementation must preserve",
				RunCtx: func(ctx cliapp.RunContext) error {
					return ctx.RenderOperational(cliapp.OperationalReport{
						Status:    Invariants(),
						NextSteps: []string{"Treat any contradiction as a release-blocking security defect."},
					})
				},
			},
		},
	}
}
