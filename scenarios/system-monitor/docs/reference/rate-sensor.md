# Disk rate sensor

System-monitor samples each configured mount and governed storage root. A
sample records bytes, the elapsed sample interval, and the positive growth
rate in bytes per hour. A root is a hot writer when its rate exceeds the
declared limit for one complete sensor window.

Pressure reports carry `fill_rate_bytes_per_hour`, `hot_writers[]`, and a typed
trigger. The receiver uses the shared storage-manager classifier; system-
monitor owns only debounce and cooldown state.

If a root cannot be measured within its budget, the sample is marked partial
and retains its unavailable or fallback trust state. A partial sample must not
be presented as a complete device census.
