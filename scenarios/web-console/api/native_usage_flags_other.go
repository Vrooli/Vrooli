//go:build !unix

package main

// Non-Unix platforms do not expose filesystem FIFOs through this flag.
const nativeNonblockingOpenFlag = 0
