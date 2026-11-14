// Package runtimeapp owns runtime-supervisor and runtime-recovery orchestration.
//
// Host lighting is enabled through the lightingruntime controller with a
// staged artifact digest and a machine-qualified device policy.
// LightingControl exposes fixed typed status/setup/enable/disable operations.
// Its owners are injectable through Service.Lighting; the concrete root and
// authenticated transport composition remains pending the scenario owner and
// machine-local permission-policy owner. Resource setup and device permission
// repair stay with their owning services. The intended scenario behavior is to
// apply saved lighting state after a backend change; no concrete ScenarioOwner
// is currently composed here.
package runtimeapp
