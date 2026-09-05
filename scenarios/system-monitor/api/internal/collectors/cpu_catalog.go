package collectors

// CPUSignal is the executable counterpart of the CPU signal catalog in the
// metric contract. Backend text is intentionally mechanism-specific so an
// unsupported observation remains useful to operators.
type CPUSignal struct {
	Key, Unit                           string
	Tier                                int
	Linux, Darwin, Windows, Unsupported string
}

var cpuSignalCatalog = []CPUSignal{
	{"usage_percent", "percent", 1, "/proc/stat", "kern.cp_time", "GetSystemTimes", "no native CPU backend"},
	{"mode_breakdown", "percent by mode", 1, "/proc/stat", "kern.cp_time", "GetSystemTimes", "no native CPU backend"},
	{"load_average", "load", 1, "/proc/loadavg", "vm.loadavg", "refuse: no Unix load average", "no load backend"},
	{"normalized_load_1", "load/core", 1, "derived from /proc/loadavg", "derived from vm.loadavg", "refuse: no load source", "no load backend"},
	{"normalized_load_5", "load/core", 1, "derived from /proc/loadavg", "derived from vm.loadavg", "refuse: no load source", "no load backend"},
	{"run_queue_depth", "processes", 1, "/proc/loadavg", "refuse: no runnable-process field", "refuse: no runnable-process counter", "no run-queue backend"},
	{"context_switches_per_second", "per second", 1, "/proc/stat ctxt", "refuse: no public counter", "refuse: PDH backend not enabled", "no counter backend"},
	{"interrupts_per_second", "per second", 1, "/proc/stat intr", "refuse: no public counter", "refuse: PDH backend not enabled", "no counter backend"},
	{"cpu_psi_some_avg10", "percent", 1, "/proc/pressure/cpu", "refuse: PSI is Linux-specific", "refuse: PSI is Linux-specific", "no PSI backend"},
	{"cpu_psi_full_avg10", "percent", 1, "/proc/pressure/cpu", "refuse: PSI is Linux-specific", "refuse: PSI is Linux-specific", "no PSI backend"},
	{"per_core_utilization", "percent/core", 2, "/proc/stat cpuN", "host_processor_info", "PDH processor instances", "no per-core backend"},
	{"core_imbalance_index", "percentage points", 2, "derived from cpuN", "derived from processor info", "derived from PDH instances", "no per-core backend"},
	{"quota_throttling", "per second, percent", 2, "cgroup v2 cpu.stat/cpu.max", "refuse: no cgroup backend", "refuse: job-object backend not enabled", "no quota backend"},
	{"frequency_derate_ratio", "ratio", 2, "sysfs scaling frequency", "refuse: no public frequency backend", "refuse: performance counter backend not enabled", "no frequency backend"},
	{"thermal_throttle_evidence", "count, celsius", 2, "hwmon and thermal zones", "refuse: thermal join not enabled", "refuse: thermal backend not enabled", "no thermal backend"},
	{"fork_rate", "per second", 3, "existing fork-rate collector", "explicit refusal if unavailable", "explicit refusal if unavailable", "no fork counter backend"},
	{"process_cpu_seconds", "seconds", 3, "process CPU ticks", "explicit refusal if unavailable", "explicit refusal if unavailable", "no process sampler backend"},
	{"historical_process_cpu", "percent", 3, "persisted process rollups", "persisted process rollups", "persisted process rollups", "no historical store"},
}
