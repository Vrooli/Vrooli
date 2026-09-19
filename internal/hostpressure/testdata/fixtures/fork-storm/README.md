# fork-storm fixture

Captured on swarminator (32 cores) at 2026-09-02T18:15:52Z by `HOSTPRESSURE_CAPTURE_FORK_STORM=1 go test ./internal/hostpressure/ -run TestCaptureForkStormFixture` during a bounded burst:

    systemd-run --user --scope --slice=vrooli-test.slice -p TasksMax=512 sh -c 'for i in $(seq 1 200); do sleep 20 & done; wait'

proc-stat-t0/t1 bracket the burst (103.698571ms apart, recorded in manifest.json); procs-t0.tsv is the process tree before it and procs.tsv the tree during it, so `hostpressure.Attribution` ranks the burst's `sh` by child count and by delta.
