# system-host-pressure

Reports host CPU pressure, memory and swap state, process count, and fork rate
through the portable `internal/hostpressure` reader. Every field carries a
read/unread state and provenance. Unread means the platform could not answer;
it is never treated as zero.

The check is diagnostic and joins into substrate SB14–SB17. Fork rate is a
Linux signal by decision and is reported unread on macOS and Windows.
