# Provider lifecycle fence

`api-core/lifecycle` is the shared transport and validation layer for the
provider-owned maintenance fence in `common.v1.LifecycleMaintenanceService`.

A control-plane replacement uses one durable operation ID to:

1. ask the live provider to close admission;
2. observe complete inventory and drain it when required;
3. retain the returned fence token and revision while replacing the process;
4. resume the provider only after the replacement is healthy.

Providers that implement the Connect service opt into this contract. A missing
service is treated as an explicitly unguarded provider; transport failures,
identity failures, incomplete inventory, and fence mismatches are hard errors.
Mutation calls are mounted behind `LoopbackOnly` because the current control
plane invokes them over loopback. A deployment that proxies a scenario must
provide an equivalent authenticated local boundary before exposing the
lifecycle service.

`emergency_override` is deliberately narrow: it may bypass drain-only blockers,
but it still requires a human/operator reason and never bypasses provider
identity, inventory validity, or fence revision checks.
