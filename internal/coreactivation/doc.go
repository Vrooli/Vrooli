// Package coreactivation owns durable root control-plane activation operations
// and once-only native effect fencing. Installer, native convergence and
// watchdog adapters supply effects; this package never builds from source,
// creates operator authority, changes schema compatibility policy or launches
// an arbitrary command. Paths come from the invoking-user runtime home.
//
// Production installer/convergence/watchdog adapters are not yet connected.
// NativeWorker supplies native fencing only; artifact qualification and actual
// persistent-state compatibility remain mandatory Effects responsibilities.
package coreactivation
