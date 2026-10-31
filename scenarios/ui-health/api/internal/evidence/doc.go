// Package evidence holds the engine-agnostic browser-evidence model and the
// verdict analysis applied to it.
//
// A UI validation phase loads a single surface (page) in a browser and observes
// what happened: whether the iframe-bridge handshake signaled, which console
// messages and failed network requests appeared, any uncaught page exceptions,
// and the storage-shim outcome. That observation set is an [Evidence] value.
// [Analyze] turns one [Evidence] into a [Verdict] (pass/fail + the single most
// significant failure message + the console/network/page-error counts).
//
// The model is deliberately free of any browser-engine transport detail so the
// same analysis serves every producer. The ui-health runtime group produces
// [Evidence] from a Browser Automation Studio (BAS) workflow timeline, while
// generic screenshot/DOM/layout findings are produced by the visual-health
// analyzer.
package evidence
