// Package setpoint is the one reader of the reliability setpoint
// (scenarios/infrastructure-manager/setpoint/reliability-setpoint.json).
//
// Every host-pressure consumer in the control plane and in vrooli-autoheal
// takes its bars from here: the emergency watchdog, the autoheal
// host-pressure check, and the runtime supervisor's pressure gate. A bar
// carries its threshold (Min or Max), its unit and its authored sustain; the
// sustain is the sustain, no consumer shortens it in code. Two bars may not
// share a cell_ref, and Load fails when they do, so the file cannot answer
// one cell two ways.
//
// When the file is unreadable, Fallback returns one documented set of
// compiled bars with Path "compiled fallback"; consumers report that source
// so a host running on fallbacks is visible.
package setpoint
