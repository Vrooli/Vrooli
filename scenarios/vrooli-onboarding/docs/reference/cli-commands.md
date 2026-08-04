# CLI Commands

`vrooli-onboarding operator show` returns effective operator state.
`vrooli-onboarding operator apply --body-file <path>` commits a state document.
`vrooli-onboarding operator scenarios` and `operator readiness` expose the same
read models used by the UI.

Use `vrooli credentials provision` with a value on standard input for direct
credential provisioning; never place a value in a CLI argument. The rest of the
credential surface is read-only or explicit: `vrooli credentials doctor`
diagnoses the host backend. `secrets-manager credentials list` prints
declarations and state without ever printing a value, and `secrets-manager
keyring inspect|repair` owns keyring operations. `vrooli credentials status
--identity <id> --field <field>` reports one credential alongside the provider
state.

See [Configuring credentials](../../../../docs/configuration/secrets.md) for the
degradation contract these commands report against.
