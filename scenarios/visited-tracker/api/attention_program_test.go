package main

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// [REQ:VT-REQ-025] Execute the authored programs against deterministic binding
// doubles to prove error classification, byte limits and unknown sensor handling.
func TestAttentionProgramEnvelopeAndFailureContracts(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal("program contract tests require Python 3: ", err)
	}
	root, err := filepath.Abs("../.vrooli/program-runtime")
	if err != nil {
		t.Fatal(err)
	}
	script := `
import contextlib, io, json, pathlib, runpy, sys, types
root = pathlib.Path(sys.argv[1])
class Handle:
    def __init__(self, rows, metadata=None): self.rows = rows; self.metadata = metadata or {"campaignRevision": "1"}
    def head(self, n): return self.rows[:n]
    def count(self): return len(self.rows)
    def meta(self): return self.metadata
def run(name, bindings, inputs):
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        runpy.run_path(str(root / (name + ".py")), init_globals=dict(bindings, inputs=inputs))
    raw = output.getvalue()
    value = json.loads(raw)
    assert len(raw.encode()) <= 65536, len(raw)
    return value
rows = [{"path": "x"*4000 + str(i), "revision": "a"*64, "fileId": str(i)} for i in range(20)]
owner = types.SimpleNamespace(attention=types.SimpleNamespace(preview=lambda **kwargs: Handle(rows)))
result = run("attention-select", {"visited_tracker": owner}, {"campaign_id": "test", "limit": 20})
assert result["status"] == "partial", result
assert result["signals"]["truncated"] is True, result
assert result["signals"]["reserved"] is False, result
assert 0 < len(result["signals"]["candidates"]) < 20, result
assert result["signals"]["candidates"][0]["path"] == rows[0]["path"]
def unreachable(**kwargs): raise RuntimeError("connection refused")
owner.attention.preview = unreachable
result = run("attention-select", {"visited_tracker": owner}, {"campaign_id": "test", "limit": 1})
assert result["status"] == "unavailable" and result["errors"][0]["class"] == "scenario_unreachable", result
result = run("attention-select", {}, {"campaign_id": "test", "limit": 1})
assert result["status"] == "failed" and result["errors"][0]["class"] == "kernel_runtime", result
for state in ("CONDITION_STATUS_UNAVAILABLE", "CONDITION_STATUS_UNSPECIFIED"):
    owner = types.SimpleNamespace(bindings=types.SimpleNamespace(condition=lambda **kwargs: Handle([{"status":state}])))
    result = run("setpoint-read", {"program_runtime":owner}, {})
    assert result["status"] == "partial", result
    row = result["signals"]["rows"][0]
    assert row["reading"] is None and row["in_band"] is None, row
owner = types.SimpleNamespace(
    bindings=types.SimpleNamespace(condition=lambda **kwargs: Handle([{"status":"CONDITION_STATUS_HEALTHY"}])),
    attention=types.SimpleNamespace(preview=lambda **kwargs: Handle([], {"observedAt":"2026-09-06T12:00:00Z", "eligibleCount": 3, "activeClaimCount": 1, "oldestEligibleAgeSeconds": "120"})),
)
result = run("setpoint-read", {"program_runtime":owner, "visited_tracker":owner}, {"campaign_id":"test"})
fairness = [row for row in result["signals"]["rows"] if row["row"] == "attention-fairness"][0]
assert fairness["reading"] == 120 and fairness["in_band"] is True, fairness
for meta, expected in (({"observedAt":"2026-09-06T12:00:00Z"}, 0),
                       ({"observedAt":"2026-09-06T12:00:00Z", "oldestEligibleAgeSeconds":"86401"}, 86401),
                       ({"observedAt":"2026-09-06T12:00:00Z", "oldestEligibleAgeSeconds":True}, None),
                       ({"observedAt":"2026-09-06T12:00:00Z", "oldestEligibleAgeSeconds":"-1"}, None),
                       ({"campaignRevision":"1"}, None)):
    owner.attention.preview = lambda **kwargs: Handle([], meta)
    result = run("setpoint-read", {"program_runtime":owner, "visited_tracker":owner}, {"campaign_id":"test"})
    fairness = [row for row in result["signals"]["rows"] if row["row"] == "attention-fairness"][0]
    assert fairness["reading"] == expected, (meta, fairness)
    assert fairness["unavailable"] is (expected is None), fairness
    assert result["status"] == "partial", result # old preview lacks integrity/storage sensors
meta = {"observedAt":"2026-09-06T12:00:00Z", "integrity":{"checkSchemaVersion":1,"observedAt":"2026-09-06T12:00:00Z","examinedClaims":2},
        "storageWrites":{"epoch":"api-one","observedAt":"2026-09-06T12:00:00Z","completedTotal":"3","failedTotal":"1","lastCompletedAt":"2026-09-06T12:00:00Z","durableSyncSupported":True}}
for patch, reading in (({},0), ({"consecutiveFailures":"1"},1), ({"lastCompletedAgeSeconds":"301"},None), ({"completedTotal":"0","failedTotal":"0"},None), ({"failedTotal":"9"},None), ({"epoch":""},None), ({"durableSyncSupported":False},None), ({"consecutiveFailures":True},None)):
    value = dict(meta, storageWrites=dict(meta["storageWrites"], **patch))
    owner.attention.preview = lambda **kwargs: Handle([], value)
    result = run("setpoint-read", {"program_runtime":owner,"visited_tracker":owner}, {"campaign_id":"test"})
    by_name = {r["row"]:r for r in result["signals"]["rows"]}
    assert by_name["durable-storage"]["reading"] == reading, (patch,result)
    assert by_name["durable-storage"]["unavailable"] is (reading is None), result
    assert by_name["claim-correctness"]["reading"] == 0, result
for patch, reading in (({"violations":1},1), ({"violations":3},None), ({"checkSchemaVersion":0},None), ({"violations":True},None), ({"observedAt":""},None)):
    value = dict(meta, integrity=dict(meta["integrity"], **patch))
    owner.attention.preview = lambda **kwargs: Handle([], value)
    result = run("setpoint-read", {"program_runtime":owner,"visited_tracker":owner}, {"campaign_id":"test"})
    row = next(r for r in result["signals"]["rows"] if r["row"] == "claim-correctness")
    assert row["reading"] == reading, (patch,row)

`
	cmd := exec.Command(python, "-c", script, root)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("program regression: %v\n%s", err, output)
	}
}
