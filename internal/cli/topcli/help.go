package topcli

const (
	StatusHelpText       = "Usage: vrooli status [--resources|--scenarios] [--fast|--no-fast] [--json]"
	DoctorHelpText       = "Usage: vrooli doctor [--json]"
	StopHelpText         = "Usage: vrooli stop [all|scenarios|resources|scenario:<name>|resource:<name>|<name>...] [--json]"
	OrphansHelpText      = "Usage: vrooli orphans [kill] [--json]"
	LocksHelpText        = "Usage: vrooli locks [clean] [--json]"
	DiagnosePortHelpText = "Usage: vrooli diagnose-port <port> [scenario] [--json]"
)
