# CLI Commands

`vrooli-onboarding operator show` returns effective operator state.
`vrooli-onboarding operator apply --body-file <path>` commits a state document.
`vrooli-onboarding operator scenarios` and `operator readiness` expose the same
read models used by the UI.

Use `vrooli credentials provision` with a value on standard input for direct
credential provisioning; never place a value in a CLI argument.
