// Package closure derives the versioned deployment closure of a scenario from
// component declarations (service.json, resource.json), the dependency
// analyzer, the shared host requirement resolver and the operator's selection.
//
// The closure is the single explanation surface: every component carries the
// reasons it is included, every platform gap is a typed unsupported entry, and
// catalog failure is a typed closure_unavailable error rather than an empty
// closure. ToSelection maps a closure onto the shared setup/v1 Selection so
// machine configuration flows through the existing onboarding contract.
package closure
