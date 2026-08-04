# CLI Commands

`vrooli-onboarding operator show` returns effective operator state.
`vrooli-onboarding operator apply --body-file <path>` commits a state document.
`vrooli-onboarding operator scenarios` and `operator readiness` expose the same
read models used by the UI.

Use `vrooli credentials provision` with a value on standard input for direct
credential provisioning; never place a value in a CLI argument. The rest of the
credential surface is read-only or explicit: `vrooli credentials doctor`
diagnoses the host backend and lists every declared credential with its
remediation, `vrooli credentials list` prints declarations and state without
ever printing a value, `vrooli credentials status --identity <id> --field
<field>` reports one credential alongside the provider state, and `vrooli
credentials delete --identity <id> --field <field> --yes` removes one.

See [Configuring credentials](../../../../docs/configuration/secrets.md) for the
degradation contract these commands report against.
