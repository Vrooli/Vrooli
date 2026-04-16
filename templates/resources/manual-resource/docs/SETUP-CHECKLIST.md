# Setup Checklist

1. Install or provision the external dependency.
2. Record any required host configuration.
3. Run the documented validation probes.
4. Capture known caveats for future operators.

Keep this checklist as the primary setup contract. Do not replace it with vague automation; only add code under `cli/internal/validate` when the validation workflow genuinely benefits from it.
