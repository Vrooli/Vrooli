# `@vrooli/ui`

Small shared browser-facing contracts that do not own provider credentials.
`auth-status.ts` is the common status/recovery model for gated UIs. A scenario
may render it with its existing component library, but must obtain status from
its same-origin API and must never parse or store `Cf-Access-Jwt-Assertion` or
the `CF_Authorization` cookie in browser code. `authenticated` describes
identity authentication; `canMutate` is the explicit human-authority
capability and must gate writes.
