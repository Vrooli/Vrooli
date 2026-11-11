#!/usr/bin/env python3
"""Tests for footprint.py. Run: python3 -I test_footprint.py (from this directory)."""

from __future__ import annotations

import io
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import footprint as fp  # noqa: E402

GIB = 1024**3
ACE = "/home/u/.vrooli/artifacts/ace-step/1.0/venv/bin/python3"
WHISPER = "/home/u/.vrooli/artifacts/whisper/1.9.2/app/whisper-server"  # key: whisper


class FakeHost:
    """Canned command outputs keyed by argv prefix. A value may be a string, (code, text) or an exception."""

    def __init__(self, gpu_procs=(), gpu_used=1000, timers=(), running=("redis",), stopped=("ace-step",),
                 containers=(), sandboxes=(), sandbox_total=None, claims=(), sizes=None, broken=()):
        self.gpu_procs, self.gpu_used = list(gpu_procs), gpu_used
        self.timers, self.running, self.stopped = list(timers), list(running), list(stopped)
        self.containers, self.sandboxes, self.claims = list(containers), list(sandboxes), list(claims)
        self.sandbox_total = sandbox_total
        self.sizes, self.broken = sizes or {}, set(broken)
        self.calls = []

    def __call__(self, argv, timeout):
        self.calls.append(list(argv))
        tool = argv[0]
        if tool in self.broken:
            raise FileNotFoundError(tool)
        if tool == "nvidia-smi" and argv[1].startswith("--query-gpu"):
            return 0, f"{self.gpu_used}, 16376\n"
        if tool == "nvidia-smi":
            return 0, "".join(f"{pid}, {name}, {mib}\n" for pid, name, mib in self.gpu_procs)
        if argv[:3] == ["vrooli", "resource", "status"]:
            rows = [{"resource": {"name": n}, "running": True} for n in self.running]
            rows += [{"resource": {"name": n}, "running": False} for n in self.stopped]
            return 0, json.dumps({"success": True, "resources": rows})
        if argv[:3] == ["vrooli", "capacity", "list"]:
            return 0, json.dumps({"claims": self.claims})
        if tool == "docker":
            return 0, "".join(f"{n}\timg:1\tUp 1 hour\n" for n in self.containers)
        if tool == "workspace-sandbox":
            rows = [f"{sid}... | {status} | reserved=/x | owner=o1 | created=2026-11-10 03:01 | files=0" for sid, status in self.sandboxes]
            return 0, json.dumps({"summary": [f"Sandboxes found: {self.sandbox_total if self.sandbox_total is not None else len(rows)}"], "results": rows})
        if tool == "systemctl":
            return 0, "".join(f"Mon 2026-11-09 22:57:56 EST 56s Mon 2026-11-09 22:56:56 EST 3s ago {u} {u[:-6]}.service\n" for u in self.timers)
        if tool == "du":
            return 0, f"{self.sizes[argv[2]]}\t{argv[2]}\n"
        raise AssertionError(f"unexpected command {argv}")


def claim(owner="ollama", kind="vram", status="granted", cid="clm-1", amount=GIB):
    return {"claim_id": cid, "owner_id": owner, "resource_kind": kind, "amount_bytes": amount, "status": status}


class FootprintCase(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.baseline = str(Path(self.tmp.name) / "base.json")

    def snapshot(self, host, *extra):
        out = io.StringIO()
        code = fp.main(["snapshot", "--out", self.baseline, "--force", *extra], run=host, out=out)
        self.assertEqual(code, 0, out.getvalue())

    def diff(self, host, *extra):
        out, err = io.StringIO(), io.StringIO()
        code = fp.main(["diff", "--baseline", self.baseline, *extra], run=host, out=out, err=err)
        return code, out.getvalue(), err.getvalue()


class GpuTests(FootprintCase):
    def test_service_restarted_with_new_pid_is_not_new(self):
        self.snapshot(FakeHost(gpu_procs=[(100, ACE, 7500)]))
        code, text, _ = self.diff(FakeHost(gpu_procs=[(999, ACE, 7500)]))
        self.assertEqual(code, 0, text)
        self.assertNotIn("NEW GPU", text)
        self.assertIn("NEW SINCE BASELINE: 0 item(s)", text)

    def test_new_gpu_process_is_reported_with_mib_and_counts(self):
        self.snapshot(FakeHost(gpu_procs=[(1, WHISPER, 1800)], gpu_used=1800))
        code, text, _ = self.diff(FakeHost(gpu_procs=[(1, WHISPER, 1800), (2, ACE, 7500)], gpu_used=9300))
        self.assertEqual(code, 1)
        self.assertIn("NEW gpu ace-step 7500 MiB", text)
        self.assertIn("used 9300 MiB, free 7076 MiB now", text)
        self.assertIn("NEW SINCE BASELINE: 1 item(s)", text)

    def test_growth_over_512_mib_of_existing_key_counts_and_is_not_reported_below(self):
        self.snapshot(FakeHost(gpu_procs=[(1, WHISPER, 1000)]))
        code, text, _ = self.diff(FakeHost(gpu_procs=[(1, WHISPER, 1700)]))
        self.assertEqual(code, 1)
        self.assertIn("GREW gpu whisper 1000 -> 1700 MiB", text)
        self.assertIn("a model was loaded into a service that was already running", text)
        self.assertIn("NEW SINCE BASELINE: 1 item(s)", text)
        code, text, _ = self.diff(FakeHost(gpu_procs=[(1, WHISPER, 1512)]))
        self.assertEqual(code, 0, text)
        self.assertNotIn("GREW", text)

    def test_second_process_with_the_same_key_counts_as_growth(self):
        self.snapshot(FakeHost(gpu_procs=[(1, WHISPER, 1000)]))
        code, text, _ = self.diff(FakeHost(gpu_procs=[(1, WHISPER, 1000), (7, WHISPER, 1200)]))
        self.assertEqual(code, 1)
        self.assertIn("GREW gpu whisper 1000 -> 2200 MiB", text)
        self.assertNotIn("NEW gpu", text)

    def test_snapshot_sums_processes_per_key(self):
        snap = fp.collect(FakeHost(gpu_procs=[(1, WHISPER, 1000), (7, WHISPER, 1200), (2, ACE, 5)]))
        self.assertEqual(snap["gpu"]["keys"], {"whisper": 2200, "ace-step": 5})

    def test_growth_is_suppressed_by_keep_file_using_the_gpu_key(self):
        self.snapshot(FakeHost(gpu_procs=[(1, WHISPER, 1000)]))
        keep = Path(self.tmp.name) / "keep.txt"
        keep.write_text("# models meant to stay loaded\n\nwhisper\n")
        code, text, _ = self.diff(FakeHost(gpu_procs=[(1, WHISPER, 3000)]), "--keep-file", str(keep))
        self.assertEqual(code, 0, text)
        self.assertIn("GREW gpu whisper 1000 -> 3000 MiB (kept)", text)
        self.assertIn("NEW SINCE BASELINE: 0 item(s)", text)

    def test_na_and_extra_spaces_are_tolerated(self):
        snap = fp.collect(FakeHost(gpu_procs=[(5, "  /usr/bin/Xorg  ", "[N/A]")]))
        proc = snap["gpu"]["processes"][0]
        self.assertEqual((proc["key"], proc["mib"]), ("Xorg", None))

    def test_key_comes_from_artifact_dir_or_basename(self):
        self.assertEqual(fp.gpu_key(ACE), "ace-step")
        self.assertEqual(fp.gpu_key("/usr/lib/xorg/Xorg"), "Xorg")
        self.assertEqual(fp.gpu_key("python3"), "python3")


class OtherSectionTests(FootprintCase):
    def test_keep_suppresses_named_timer_by_prefix_and_exact(self):
        self.snapshot(FakeHost())
        host = FakeHost(timers=["research-freelance-opportunity.timer"])
        code, text, _ = self.diff(host)
        self.assertEqual(code, 1)
        self.assertIn("NEW timer research-freelance-opportunity.timer", text)
        self.assertIn("systemctl --user stop research-freelance-opportunity.timer", text)
        for keep in ("research-freelance", "research-freelance-opportunity.timer"):
            code, text, _ = self.diff(host, "--keep", keep)
            self.assertEqual(code, 0, text)
            self.assertIn("NEW SINCE BASELINE: 0 item(s)", text)

    def test_resource_stopped_now_is_not_reported_and_newly_running_is(self):
        self.snapshot(FakeHost(running=["redis", "ace-step"], stopped=["qdrant"]))
        code, text, _ = self.diff(FakeHost(running=["redis"], stopped=["ace-step", "qdrant"]))
        self.assertEqual(code, 0, text)
        self.assertNotIn("ace-step", text)
        code, text, _ = self.diff(FakeHost(running=["redis", "qdrant"], stopped=["ace-step"]))
        self.assertEqual(code, 1)
        self.assertIn("NEW resource qdrant", text)
        self.assertIn("vrooli resource stop qdrant", text)

    def test_new_claim_and_container_count_with_hint(self):
        self.snapshot(FakeHost(claims=[claim()], containers=["pg"]))
        host = FakeHost(
            claims=[claim(), claim(owner="ace-step", cid="clm-9", amount=8 * GIB), claim(owner="x", status="released", cid="clm-0")],
            containers=["pg", "comfy"])
        code, text, _ = self.diff(host)
        self.assertEqual(code, 1)
        self.assertIn("NEW claim clm-9 ", text)
        self.assertIn("vrooli capacity release clm-9", text)
        self.assertIn("NEW container comfy ", text)
        self.assertIn("docker stop comfy", text)
        self.assertNotIn("clm-0", text)
        self.assertIn("NEW SINCE BASELINE: 2 item(s)", text)

    def test_sandboxes_are_never_listed_or_counted_even_when_ids_differ(self):
        self.snapshot(FakeHost(sandboxes=[("aaa", "active")], sandbox_total=200))
        code, text, _ = self.diff(FakeHost(sandboxes=[("bbb", "active"), ("ccc", "active")], sandbox_total=230))
        self.assertEqual(code, 0, text)
        self.assertIn("sandboxes: 230 active now, 200 at baseline, +30", text)
        self.assertNotIn("bbb", text)
        self.assertNotIn("NEW sandbox", text)
        self.assertIn("NEW SINCE BASELINE: 0 item(s)", text)

    def test_diff_only_runs_read_only_commands(self):
        host = FakeHost()
        self.snapshot(host)
        host2 = FakeHost(timers=["a.timer"], containers=["c"])
        self.diff(host2)
        self.assertTrue(all(c[0] in ("nvidia-smi", "vrooli", "docker", "workspace-sandbox", "systemctl", "du") for c in host2.calls))
        self.assertFalse(any(len(c) > 2 and c[1] in ("stop", "release") for c in host2.calls))
        self.assertFalse(any(c[:2] == ["vrooli", "resource"] and c[2] in ("stop", "start") for c in host2.calls))


class FailureAndExitTests(FootprintCase):
    def test_failing_collector_is_not_checked_and_never_new(self):
        self.snapshot(FakeHost(containers=["pg"]))
        code, text, _ = self.diff(FakeHost(broken=["docker", "nvidia-smi"]))
        self.assertEqual(code, 3, text)
        self.assertIn("NOT CHECKED containers: ", text)
        self.assertIn("NOT CHECKED gpu: ", text)
        self.assertIn("NEW SINCE BASELINE: 0 item(s); 2 section(s) not checked", text)

    def test_collectors_tolerate_nonzero_exit_timeout_and_garbage(self):
        def run(argv, timeout):
            if argv[0] == "docker":
                raise subprocess.TimeoutExpired(argv, timeout)
            if argv[0] == "systemctl":
                return 1, ""
            return 0, "not json"
        snap = fp.collect(run)
        for section in ("gpu", "resources", "claims", "containers", "sandboxes", "timers"):
            self.assertFalse(snap[section]["available"], section)
            self.assertTrue(snap[section]["error"])
        self.assertTrue(snap["disk"]["available"])

    def test_unavailable_in_baseline_only_is_never_new(self):
        self.snapshot(FakeHost(broken=["docker"]))
        code, text, _ = self.diff(FakeHost(containers=["fresh"]))
        self.assertEqual(code, 0, text)
        self.assertIn("NOT CHECKED containers: baseline: docker not found", text)
        self.assertNotIn("NEW container", text)

    def test_watched_dir_growth_over_flag_is_flagged_but_exit_is_zero(self):
        d = "/data/images"
        self.snapshot(FakeHost(sizes={d: 10 * GIB}), "--watch", d)
        code, text, _ = self.diff(FakeHost(sizes={d: 10 * GIB + fp.DISK_FLAG_BYTES + GIB}), "--disk")
        self.assertEqual(code, 0, text)
        self.assertIn(f"DISK watch {d}", text)
        self.assertIn("FLAG", text)
        code, text, _ = self.diff(FakeHost(sizes={d: 11 * GIB}), "--disk")
        self.assertNotIn("FLAG", text)

    def test_diff_without_disk_flag_never_calls_du(self):
        d = "/data/images"
        self.snapshot(FakeHost(sizes={d: GIB}), "--watch", d)
        host = FakeHost(sizes={d: 50 * GIB})
        code, text, _ = self.diff(host, "--watch", d)
        self.assertEqual(code, 0, text)
        self.assertFalse([c for c in host.calls if c[0] == "du"])
        self.assertIn("DISK free", text)
        self.assertNotIn("DISK watch /", text)

    def test_diff_with_disk_remeasures_baseline_watch_dirs(self):
        d = "/data/images"
        self.snapshot(FakeHost(sizes={d: GIB}), "--watch", d)
        host = FakeHost(sizes={d: GIB})
        _, text, _ = self.diff(host, "--disk")
        self.assertIn(["du", "-sb", d], host.calls)
        self.assertIn(f"DISK watch {d}", text)

    def test_snapshot_refuses_to_overwrite_without_force(self):
        self.snapshot(FakeHost(containers=["old"]))
        err = io.StringIO()
        code = fp.main(["snapshot", "--out", self.baseline], run=FakeHost(containers=["new"]), out=io.StringIO(), err=err)
        self.assertEqual(code, 2)
        self.assertIn("already exists", err.getvalue())
        self.assertEqual(len(err.getvalue().strip().splitlines()), 1)
        self.assertEqual(json.loads(Path(self.baseline).read_text())["containers"]["items"][0]["name"], "old")
        self.assertEqual(fp.main(["snapshot", "--out", self.baseline, "--force"], run=FakeHost(containers=["new"]), out=io.StringIO()), 0)
        self.assertEqual(json.loads(Path(self.baseline).read_text())["containers"]["items"][0]["name"], "new")

    def test_snapshot_to_a_new_path_needs_no_force(self):
        fresh = str(Path(self.tmp.name) / "sub" / "fresh.json")
        self.assertEqual(fp.main(["snapshot", "--out", fresh], run=FakeHost(), out=io.StringIO()), 0)
        self.assertTrue(Path(fresh).exists())

    def test_keep_file_parsing_comments_blanks_and_missing_file(self):
        keep = Path(self.tmp.name) / "keep.txt"
        keep.write_text("# intended\n\n  research-freelance  \n   \n#timer-x\ncomfy\n")
        self.assertEqual(fp.read_keep_file(keep), ["research-freelance", "comfy"])
        self.assertEqual(fp.read_keep_file(Path(self.tmp.name) / "absent.txt"), [])
        self.snapshot(FakeHost())
        host = FakeHost(timers=["research-freelance-opportunity.timer", "timer-x.timer"], containers=["comfy"])
        code, text, _ = self.diff(host, "--keep-file", str(keep))
        self.assertEqual(code, 1)
        self.assertIn("NEW timer timer-x.timer", text)
        self.assertNotIn("NEW timer timer-x.timer (kept)", text)
        self.assertIn("NEW SINCE BASELINE: 1 item(s)", text)
        code, text, _ = self.diff(FakeHost(timers=["a.timer"]), "--keep-file", str(Path(self.tmp.name) / "absent.txt"))
        self.assertEqual(code, 1)

    def test_keep_and_keep_file_combine(self):
        self.snapshot(FakeHost())
        keep = Path(self.tmp.name) / "keep.txt"
        keep.write_text("a.timer\n")
        code, text, _ = self.diff(FakeHost(timers=["a.timer", "b.timer"]), "--keep-file", str(keep), "--keep", "b.timer")
        self.assertEqual(code, 0, text)

    def test_gpu_or_resources_not_checked_exits_3_with_not_checked_line(self):
        self.snapshot(FakeHost())
        code, text, _ = self.diff(FakeHost(broken=["nvidia-smi"]))
        self.assertEqual(code, 3)
        self.assertIn("NOT CHECKED gpu: nvidia-smi not found", text)
        self.assertIn("NEW SINCE BASELINE: 0 item(s); 1 section(s) not checked", text)
        code, text, _ = self.diff(FakeHost(broken=["vrooli"]))
        self.assertEqual(code, 3)
        self.assertIn("NOT CHECKED resources:", text)

    def test_other_section_not_checked_exits_0(self):
        self.snapshot(FakeHost())
        code, text, _ = self.diff(FakeHost(broken=["systemctl"]))
        self.assertEqual(code, 0, text)
        self.assertIn("NOT CHECKED timers:", text)

    def test_new_item_wins_over_not_checked_gpu(self):
        self.snapshot(FakeHost())
        code, text, _ = self.diff(FakeHost(broken=["nvidia-smi"], timers=["a.timer"]))
        self.assertEqual(code, 1)
        self.assertIn("NEW SINCE BASELINE: 1 item(s); 1 section(s) not checked", text)

    def test_unreadable_baseline_exits_2(self):
        err = io.StringIO()
        self.assertEqual(fp.main(["diff", "--baseline", self.baseline], run=FakeHost(), out=io.StringIO(), err=err), 2)
        Path(self.baseline).write_text("{not json")
        self.assertEqual(fp.main(["diff", "--baseline", self.baseline], run=FakeHost(), out=io.StringIO(), err=err), 2)
        Path(self.baseline).write_text("[]")
        self.assertEqual(fp.main(["diff", "--baseline", self.baseline], run=FakeHost(), out=io.StringIO(), err=err), 2)

    def test_usage_error_exits_2(self):
        with contextlib_redirect_stderr():
            self.assertEqual(fp.main(["diff"], run=FakeHost()), 2)
            self.assertEqual(fp.main([], run=FakeHost()), 2)

    def test_json_output_parses_and_carries_the_same_count(self):
        self.snapshot(FakeHost(gpu_procs=[(1, WHISPER, 1800)]))
        host = FakeHost(gpu_procs=[(1, WHISPER, 1800), (2, ACE, 7500)], timers=["t.timer"])
        code, text, _ = self.diff(host)
        jcode, jtext, _ = self.diff(host, "--json")
        data = json.loads(jtext)
        self.assertEqual((code, jcode), (1, 1))
        self.assertEqual(data["new_count"], 2)
        self.assertEqual(data["not_checked"], [])
        self.assertEqual(data["exit_code"], 1)
        self.assertIn("NEW SINCE BASELINE: 2 item(s)", text)
        self.assertEqual(data["gpu"]["new"][0]["key"], "ace-step")
        self.assertEqual(data["timers"]["new"][0]["unit"], "t.timer")

    def test_json_lists_not_checked_sections(self):
        self.snapshot(FakeHost())
        code, text, _ = self.diff(FakeHost(broken=["nvidia-smi"]), "--json")
        data = json.loads(text)
        self.assertEqual(code, 3)
        self.assertEqual([n["section"] for n in data["not_checked"]], ["gpu"])

    def test_snapshot_records_all_sections_and_utc_timestamp(self):
        self.snapshot(FakeHost(gpu_procs=[(1, ACE, 7500)], timers=["t.timer"]))
        snap = json.loads(Path(self.baseline).read_text())
        for section in ("gpu", "resources", "claims", "containers", "sandboxes", "timers", "disk"):
            self.assertIn(section, snap)
        self.assertTrue(snap["taken_at"].endswith("Z"))
        self.assertEqual(snap["gpu"]["total_mib"], 16376)
        self.assertEqual(snap["disk"]["watched"], {})
        self.assertEqual(snap["sandboxes"], {"available": True, "total_active": 0})
        self.assertGreater(snap["disk"]["home_free_bytes"], 0)


class contextlib_redirect_stderr:
    def __enter__(self):
        self.saved, sys.stderr = sys.stderr, io.StringIO()

    def __exit__(self, *exc):
        sys.stderr = self.saved


if __name__ == "__main__":
    unittest.main()
