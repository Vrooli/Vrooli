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

One sample is also bounded as a whole. A total budget (2 s by default) caps the
filesystem walking of every root in one tick; a root's own budget is clipped to
what remains, and roots the total cannot reach are omitted from that sample
rather than reported as zero bytes. A root that expands to its child
directories measures a fixed number of children per tick (4 by default),
rotating through them across ticks, so a child is measured every
`ceil(children / 4)` intervals. Each snapshot carries its own observation time
and elapsed hours, so the rate estimate stays honest about that spacing; the
declared `sample_interval_seconds` remains the tick cadence.
