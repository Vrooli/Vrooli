// Package logvolumebounds is the host safeguard that makes the flat-file system
// log store (the files rsyslog writes under /var/log and logrotate rotates)
// unable to fill the disk, whatever is written to it.
//
// # Why it exists
//
// On 2026-09-01 the child process of gnome-keyring-daemon reached its 1,024
// file-descriptor limit, its accept loop began failing on every iteration, and
// rsyslog wrote the failure line to /var/log/syslog and /var/log/auth.log about
// 50,000 times a second. The two files reached 320 GB in 70 hours and the root
// filesystem hit 100% twice. journald, which caps itself at 4 GB by default,
// was unaffected. The flat files had a weekly rotation with no size trigger,
// so nothing could act for up to seven days.
//
// Fixing the writer is a separate concern (see internal/credentials and the
// keyring safeguards). This safeguard is defence in depth: the next runaway
// writer will be a different process, and a log store must be bounded by
// construction.
//
// # What it changes
//
//   - /etc/logrotate.d/rsyslog gains a `maxsize` directive (default 1G) in every
//     block that does not already bound itself. The distribution's file list,
//     schedule, rotate count and scripts are kept verbatim; the original is
//     preserved once at /var/lib/vrooli/log-volume-bounds/logrotate-rsyslog.orig.
//     The edit is in place because logrotate rejects two stanzas for one file.
//   - /etc/systemd/system/logrotate.timer.d/99-vrooli-hourly.conf runs logrotate
//     hourly, so `maxsize` is checked within the hour instead of once a day.
//   - /etc/rsyslog.d/05-vrooli-ratelimit.conf sets a per-process ceiling on the
//     system log socket (default 1000 messages per 5 seconds). The excess is
//     dropped and rsyslog records one summary line. journald forwards the
//     original sender credentials, so the limit applies per faulting process.
//   - Emergency reclaim: when a bounded file is already more than
//     `emergency_multiplier` (default 8) times the bound, Apply preserves its
//     last megabyte under /var/log/vrooli/log-volume-bounds/ and truncates it.
//     Bytes beyond the retention bound are unrecoverable evidence anyway, and a
//     forced rotation would only move them to `.1` without freeing space.
//
// # Portability
//
// Linux with systemd, rsyslog and logrotate. Hosts without a flat rsyslog
// stanza report not-applicable rather than pending: journald-only hosts are
// already bounded. macOS bounds its unified log store and newsyslog by default
// and Windows caps each event log; neither has this failure mode.
package logvolumebounds
