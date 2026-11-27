#!/usr/bin/env python3
"""Performance Health producer for LM1 mixed admission (explicit live workload)."""
from __future__ import annotations

import hashlib
import json
import math
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import threading
import time
import uuid
from datetime import datetime, timezone
from urllib.request import urlopen

ROOT = pathlib.Path(__file__).resolve().parents[3]
SCENARIO = ROOT / "scenarios/research-desk"
EVIDENCE = pathlib.Path.home() / ".vrooli/research/topics/local-model-performance/runs/local-coding-runtime-qualification-20261124/evidence"
AUDIO = ROOT / "scenarios/audio-tools/bas/fixtures/dictation-reference.wav"
PROMPT = "a small red cube on a neutral tabletop"
GPU_FLOOR, GPU_HEADROOM = 6 << 30, 1 << 30
RAM_FLOOR, RAM_HEADROOM = 8 << 30, 2 << 30
QUERIES = [
    "local model policy roles and router configuration",
    "audio transcription local whisper provider latency",
    "embedding migration workflow qdrant collection dimensions",
    "capacity broker GPU no-reclaim admission",
    "image tools diffusers instruct pix2pix GPU backend",
    "model residency unload stale claim reconciliation",
    "search hub query caller real software documentation",
    "host free memory and swap monitoring operating procedure",
    "local model rights OpenRAIL commercial use obligations",
]

def sha(b: bytes) -> str:
    return hashlib.sha256(b).hexdigest()

def run(args: list[str], timeout: int = 30, cwd: pathlib.Path = ROOT) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, cwd=cwd, capture_output=True, text=True, timeout=timeout, check=False)

def json_stdout(result: subprocess.CompletedProcess[str]):
    if result.returncode:
        raise RuntimeError(f"{result.args[0]} exited {result.returncode}: {result.stderr[-1000:]}")
    return json.loads(result.stdout)

def meminfo() -> dict[str, int]:
    rows = {}
    for line in pathlib.Path("/proc/meminfo").read_text().splitlines():
        key, _, rest = line.partition(":")
        vals = rest.split()
        if vals and vals[0].isdigit():
            rows[key] = int(vals[0])
    return rows

def oom_count() -> int:
    for line in pathlib.Path("/proc/vmstat").read_text().splitlines():
        if line.startswith("oom_kill "):
            return int(line.split()[1])
    raise RuntimeError("host OOM counter unavailable")

def workload_identity(output: pathlib.Path):
    invocation = json.loads((output.parent / "invocation.json").read_text())
    d = invocation["definition"]
    raw = bytearray()
    for rel in sorted(d["sources"]):
        raw.extend(rel.replace(os.sep, "/").encode() + b"\0")
        raw.extend((SCENARIO / rel).read_bytes() + b"\0")
    contract = (SCENARIO / d["contract"]).read_bytes()
    # Match Go encoding/json for workload.Definition, whose fields are struct ordered.
    p95 = float(d["p95_max_ms"])
    d["p95_max_ms"] = int(p95) if p95.is_integer() else p95
    d["min_cpu_cores"] = int(d.get("min_cpu_cores", 0))
    d["min_memory_bytes"] = int(d.get("min_memory_bytes", 0))
    config_bytes = json.dumps(d, separators=(",", ":"), ensure_ascii=False).encode()
    return invocation, sha(bytes(raw)), sha(contract), sha(config_bytes)

def candidate(api_url: str) -> dict:
    with urlopen(api_url.rstrip("/") + "/health", timeout=5) as res:
        body = json.loads(res.read(1 << 20))
    build = body.get("build_identity", "")
    if not isinstance(build, str) or not build.startswith("sha256:") or len(build) != 71:
        raise RuntimeError("target health has no managed build identity")
    return {"build_identity": build, "driver_version": "", "browser_version": ""}

def caller_commands():
    cli = ROOT / ".vrooli/bin"
    out = []
    for i in range(10):
        out.append((f"audio-{i+1:02d}", [str(cli/"audio-tools"), "stt", "transcribe", "--file", str(AUDIO), "--language", "en", "--json"]))
    for i, q in enumerate(QUERIES, 1):
        out.append((f"search-{i:02d}", [str(cli/"search-hub"), "query", "query", q, "--type", "command", "--limit", "1", "--json"]))
    out.append(("gateway-generate-01", [
        str(cli/"ai-gateway"), "routing", "execute", "--kind", "text", "--role", "chat.small",
        "--profile", "local-only", "--privacy-class", "internal", "--privacy-provenance",
        "research-desk.lm1_unit3_mixed_showcase", "--operation", "generate", "--scenario", "research-desk",
        "--max-cost-usd", "0", "--max-output-tokens", "32", "--timeout-ms", "30000", "--json",
        "--input-text", "Return exactly: local gateway caller completed.",
    ]))
    return out

def image_command(output: pathlib.Path, operation_id: str) -> list[str]:
    return [
        str(ROOT/".vrooli/bin/image-tools"), "ai", "generate", "--model", "sd-1.5",
        "--prompt", PROMPT, "--negative", "text, watermark, logo", "--width", "256",
        "--height", "256", "--seed", "42", "--operation-id", operation_id,
        "--priority", "batch", "--no-reclaim", "--require-gpu", "--fallback-policy",
        "local_only", "--wait", "--out", str(output), "--json",
    ]

def success(row: dict) -> bool:
    try:
        b = json.loads(row["stdout"])
    except (ValueError, TypeError):
        return False
    if not isinstance(b, dict) or row["returncode"] != 0:
        return False
    n = row["name"]
    if n.startswith("audio-"):
        return bool(str(b.get("text", "")).strip()) and b.get("providerTier") == "PROVIDER_TIER_LOCAL"
    if n.startswith("search-"):
        return any(isinstance(g, dict) and g.get("hits") for g in b.get("groups", []))
    return n == "gateway-generate-01" and b.get("valid") is True and b.get("evidence", {}).get("status") == "success"

def cleanup(procs):
    for p in procs:
        if p.poll() is None:
            p.terminate()
    for p in procs:
        if p.poll() is None:
            try:
                p.wait(timeout=4)
            except subprocess.TimeoutExpired:
                p.kill(); p.wait()

def main(output_arg: str, api_url: str) -> int:
    output = pathlib.Path(output_arg).resolve()
    output.mkdir(mode=0o700, parents=True, exist_ok=False)
    invocation, producer_digest, contract_digest, _ = workload_identity(output)
    before = candidate(api_url)
    baseline = EVIDENCE / "LM1-PRIMARY-CHECK-BASELINE-20261125T1710Z.json"
    if not baseline.is_file():
        raise RuntimeError("LM1 cohort footprint baseline is unavailable")
    footprint = run(["python3", "-I", str(ROOT/"scenarios/research-desk/skills/research-coordination/scripts/footprint.py"),
                     "diff", "--baseline", str(baseline), "--disk", "--watch", str(output)], 120)
    if footprint.returncode:
        raise RuntimeError("cohort footprint denies execution: " + footprint.stdout[-1000:])
    cap = json_stdout(run([str(ROOT/".vrooli/build/vrooli"), "capacity", "list", "--json"]))
    unsettled = [c for c in cap.get("claims", []) if isinstance(c, dict)
                 and c.get("resource_kind") == "vram"
                 and c.get("status") not in ("granted", "released", "expired", "preempted")]
    if unsettled:
        raise RuntimeError("unsettled VRAM claims exist; no claim will be stopped or released")
    jobs = json_stdout(run([str(ROOT/".vrooli/bin/image-tools"), "jobs", "list", "--limit", "100", "--json"]))
    if any(j.get("state") in ("JOB_STATE_QUEUED", "JOB_STATE_RUNNING") for j in jobs.get("jobs", []) if isinstance(j, dict)):
        raise RuntimeError("Image Tools queue is not empty")
    disk_rows = json_stdout(run([str(ROOT/".vrooli/build/vrooli"), "capacity", "disk-list", "--json"]))
    if not isinstance(disk_rows, list) or any(isinstance(d, dict) and d.get("status") == "active"
            and str(d.get("owner_id", "")).startswith("image-tools:") for d in disk_rows):
        raise RuntimeError("Image Tools has an active disk lease")
    audio = json_stdout(run([str(ROOT/".vrooli/bin/audio-tools"), "settings", "provider", "--json"]))
    audio_config = audio.get("config", {})
    if (audio_config.get("localEnabled") is not True or audio_config.get("byokEnabled") is not False
            or audio_config.get("vrooliEnabled") is not False):
        raise RuntimeError("Audio Tools must be local-only")
    gpu = run(["nvidia-smi", "--query-gpu=memory.free", "--format=csv,noheader,nounits"], 5)
    gpu_rows = [x.strip() for x in gpu.stdout.splitlines() if x.strip()]
    if gpu.returncode or len(gpu_rows) != 1 or not gpu_rows[0].isdigit():
        raise RuntimeError("unambiguous GPU inventory unavailable")
    gpu_free = int(gpu_rows[0])
    ram_free = meminfo().get("MemAvailable", 0) * 1024
    if gpu_free * (1 << 20) < GPU_FLOOR + GPU_HEADROOM or ram_free < RAM_FLOOR + RAM_HEADROOM:
        raise RuntimeError("GPU/RAM admission floor and safety headroom not met")
    explain = json_stdout(run([
        str(ROOT/".vrooli/bin/image-tools"), "ai", "generate", "--model", "sd-1.5", "--prompt", PROMPT,
        "--negative", "text, watermark, logo", "--width", "256", "--height", "256", "--seed", "42",
        "--priority", "batch", "--no-reclaim", "--require-gpu", "--fallback-policy", "local_only",
        "--explain", "--json",
    ], 60))
    if explain.get("tier") != "local-gpu" or explain.get("gpu_viable") is not True:
        raise RuntimeError("fixed image request has no normal GPU admission path")

    overall = time.monotonic()
    started_at = datetime.now(timezone.utc)
    swap_before = meminfo().get("SwapTotal", 0) - meminfo().get("SwapFree", 0)
    oom_before = oom_count()
    stop = threading.Event()
    samples = []
    def monitor():
        while not stop.is_set():
            row = {"t_s": round(time.monotonic()-overall, 3)}
            try:
                g = run(["nvidia-smi", "--query-gpu=memory.used,memory.free,utilization.gpu", "--format=csv,noheader,nounits"], 5)
                row.update(gpu=g.stdout.strip(), gpu_rc=g.returncode,
                           swap_used_kib=meminfo().get("SwapTotal", 0)-meminfo().get("SwapFree", 0),
                           oom_kills=oom_count())
            except Exception as e:
                row["sample_error"] = str(e)
            samples.append(row)
            stop.wait(.25)
    thread = threading.Thread(target=monitor, daemon=True); thread.start()
    commands = caller_commands()
    temp = pathlib.Path(tempfile.mkdtemp(prefix="lm1-mixed-"))
    gate = temp / "go"
    ready = [temp / f"{i}.ready" for i in range(20)]
    began = [temp / f"{i}.started" for i in range(20)]
    code = ("import os,pathlib,sys,time; g=pathlib.Path(sys.argv[1]); r=pathlib.Path(sys.argv[2]); "
            "s=pathlib.Path(sys.argv[3]); a=sys.argv[4:]; r.touch(); exec('while not g.exists(): time.sleep(.002)'); "
            "s.write_text(str(time.monotonic())); os.execv(a[0],a)")
    procs = []
    exit_times = {}
    exit_lock = threading.Lock()
    waiter_by_owner = {}
    image_proc = None
    caller_rows = []
    image_path = output/"image.png"
    image_cmd = image_command(image_path, str(uuid.UUID(hex=output.parent.name)))
    try:
        for i, (name, argv) in enumerate(commands):
            out = (output/f"{name}.stdout").open("w")
            err = (output/f"{name}.stderr").open("w")
            p = subprocess.Popen([sys.executable, "-c", code, str(gate), str(ready[i]), str(began[i]), *argv],
                                 cwd=ROOT, stdout=out, stderr=err, text=True)
            procs.append((name, argv, p, out, err))
            def note_exit(proc=p, owner=name):
                proc.wait()
                with exit_lock:
                    exit_times[owner] = time.monotonic()
            waiter = threading.Thread(target=note_exit, daemon=True)
            waiter.start()
            waiter_by_owner[name] = waiter
        deadline = time.monotonic()+10
        while not all(x.exists() for x in ready) and time.monotonic() < deadline: time.sleep(.01)
        if not all(x.exists() for x in ready): raise RuntimeError("20-client barrier readiness failed")
        gate.touch()
        deadline = time.monotonic()+10
        while not all(x.exists() for x in began) and time.monotonic() < deadline: time.sleep(.01)
        if not all(x.exists() for x in began): raise RuntimeError("not all real callers started")
        for name, argv, p, out, err in procs:
            if p.poll() is not None: raise RuntimeError(f"{name} ended before image submission; overlap unproved")
        if len(procs) != 20: raise RuntimeError("fixed caller denominator changed")
        image_start = time.monotonic()
        image_proc = subprocess.Popen(image_cmd, cwd=ROOT, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        caller_intervals = {}
        for name, argv, p, out, err in procs:
            try: p.wait(timeout=240)
            except subprocess.TimeoutExpired: p.kill(); p.wait()
            waiter_by_owner[name].join(timeout=1)
            out.close(); err.close()
            with exit_lock:
                end_mono = exit_times.get(name, time.monotonic())
            start_mono = float(began[commands.index((name, argv))].read_text())
            caller_intervals[name] = (start_mono, end_mono)
            caller_rows.append({"name": name, "command": argv, "returncode": p.returncode,
                                "start_offset_s": start_mono-overall, "end_offset_s": end_mono-overall,
                                "stdout_path": f"{name}.stdout", "stderr_path": f"{name}.stderr",
                                "stdout": (output/f"{name}.stdout").read_text(errors="replace"),
                                "stderr": (output/f"{name}.stderr").read_text(errors="replace")})
        try: image_stdout, image_stderr = image_proc.communicate(timeout=300)
        except subprocess.TimeoutExpired: image_proc.kill(); image_stdout, image_stderr = image_proc.communicate()
    finally:
        cleanup([x[2] for x in procs] + ([image_proc] if image_proc else []))
        for _, _, _, out, err in procs:
            if not out.closed: out.close()
            if not err.closed: err.close()
        stop.set(); thread.join(timeout=3); shutil.rmtree(temp, ignore_errors=True)
    image_row = {"name":"image-tools-gpu-generation", "command":image_cmd, "returncode":image_proc.returncode,
                 "start_offset_s":image_start-overall, "end_offset_s":time.monotonic()-overall,
                 "stdout":image_stdout, "stderr":image_stderr,
                 "application_success":image_proc.returncode == 0 and image_path.is_file()}
    if image_path.is_file():
        image_bytes=image_path.read_bytes()
        image_row["output_artifact"]={"path":str(image_path),"sha256":sha(image_bytes),"bytes":len(image_bytes)}
    try: image_body = json.loads(image_stdout)
    except ValueError: image_body = {}
    job_id = image_body.get("job_id") or image_body.get("jobId") or ""
    job_result = run([str(ROOT/".vrooli/bin/image-tools"), "jobs", "get", str(job_id), "--json"], 30) if job_id else None
    try: job_receipt = json_stdout(job_result) if job_result else {}
    except Exception: job_receipt = {"returncode":job_result.returncode,"stdout":job_result.stdout[-2000:]}
    job = job_receipt.get("job",{}) if isinstance(job_receipt,dict) else {}
    def job_offset(field: str):
        value = job.get(field) if isinstance(job, dict) else None
        if not value:
            return None
        try:
            wall = datetime.fromisoformat(str(value).replace("Z", "+00:00"))
            return overall + (wall - started_at).total_seconds()
        except (TypeError, ValueError):
            return None
    job_start_mono, job_end_mono = job_offset("started_at"), job_offset("finished_at")
    meta = job.get("result_meta",{}) if isinstance(job,dict) else {}
    claim_id = meta.get("gpu_capacity_claim_id") if isinstance(meta,dict) else None
    post = run([str(ROOT/".vrooli/build/vrooli"),"capacity","list","--all","--json"],30)
    try: claims_after = json_stdout(post)
    except Exception: claims_after = {}
    claim_rows = claims_after.get("claims",[]) if isinstance(claims_after,dict) else []
    matching = [c for c in claim_rows if isinstance(c,dict) and c.get("claim_id")==claim_id]
    disk_receipt = {}
    disk_id = meta.get("disk_job_reservation_id") if isinstance(meta,dict) else None
    if disk_id:
        dr = run([str(ROOT/".vrooli/build/vrooli"),"capacity","disk-list","--json"],30)
        try:
            disk_rows = json_stdout(dr)
            disk_receipt = next((r for r in disk_rows if isinstance(r,dict) and r.get("id")==disk_id),{})
        except Exception: disk_receipt = {"returncode":dr.returncode,"stdout":dr.stdout[-1000:]}
    swap_after = meminfo().get("SwapTotal",0)-meminfo().get("SwapFree",0)
    oom_after = oom_count()
    valid = [s for s in samples if s.get("gpu_rc")==0 and isinstance(s.get("swap_used_kib"),int) and isinstance(s.get("oom_kills"),int)]
    during_job = [s for s in valid if job_start_mono is not None and job_end_mono is not None
                  and job_start_mono <= overall+s["t_s"] <= job_end_mono]
    swap_peak = max((s["swap_used_kib"] for s in during_job),default=swap_before)
    expected_samples = max(3, math.ceil(max(0.0, (job_end_mono-job_start_mono) if job_start_mono is not None and job_end_mono is not None else 0.0) * 3))
    counts = {"audio-tools":sum(success(r) for r in caller_rows if r["name"].startswith("audio-")),
              "search-hub":sum(success(r) for r in caller_rows if r["name"].startswith("search-")),
              "ai-gateway":sum(success(r) for r in caller_rows if r["name"].startswith("gateway-"))}
    gates = {
        "20_real_callers_started":len(caller_rows)==20,
        "all_callers_succeeded":sum(counts.values())==20,
        "all_callers_overlap_image_job": job_start_mono is not None and job_end_mono is not None
            and len(caller_intervals)==20 and all(start <= job_start_mono <= end for start,end in caller_intervals.values()),
        "image_succeeded":job.get("state")=="JOB_STATE_SUCCEEDED" and image_row["application_success"],
        "image_job_gpu_admitted":job.get("lane")=="JOB_LANE_GPU" and bool(claim_id) and bool(meta.get("gpu_uuid")),
        "image_gpu_claim_released":post.returncode==0 and any(c.get("status")=="released" for c in matching)
             and not any(c.get("status") not in ("released","expired","preempted") for c in matching)
             and meta.get("capacity_release")=="released",
        "disk_lease_completed":disk_receipt.get("status")=="completed" and disk_receipt.get("cleanup_verified") is True,
        "no_swap_growth":len(during_job)>=expected_samples and swap_peak<=swap_before and swap_after<=swap_before,
        "no_oom":oom_after==oom_before and all(s.get("oom_kills")==oom_before for s in during_job),
        "monitor_coverage":job_start_mono is not None and job_end_mono is not None and len(during_job)>=expected_samples,
    }
    gates["passed"] = all(gates.values())
    report = {"preflight":{"footprint":footprint.stdout,"capacity_before":cap,
                           "gpu_free_mib":gpu_free,"ram_available_bytes":ram_free,"image_tools_explain":explain,
                           "audio_provider":audio_config},
              "caller_success_counts":counts,"callers":caller_rows,"image_result":image_row,"image_job":job_receipt,
              "capacity_receipt":{"before":cap,"after":claims_after,"claim_id":claim_id,"matching":matching},
              "disk_receipt":disk_receipt,
              "host_receipt":{"samples":samples,"swap_before_kib":swap_before,"swap_after_kib":swap_after,
                              "swap_peak_kib":swap_peak,"oom_before":oom_before,"oom_after":oom_after},
              "gate":gates}
    (output/"showcase.json").write_text(json.dumps(report,indent=2,sort_keys=True)+"\n")
    owner_items = {"audio-tools":[r for r in caller_rows if r["name"].startswith("audio-")],
                   "search-hub":[r for r in caller_rows if r["name"].startswith("search-")],
                   "ai-gateway":[r for r in caller_rows if r["name"].startswith("gateway-")],
                   "image-tools":[image_row],"platform-capacity":[report["capacity_receipt"]],
                   "host-observation":[report["host_receipt"]]}
    artifacts=[]
    for owner, items in owner_items.items():
        filename=owner.replace("-","_")+".json"
        payload=(json.dumps({"owner":owner,"items":items},indent=2,sort_keys=True)+"\n").encode()
        (output/filename).write_bytes(payload)
        artifacts.append({"type":owner,"path":str(output/filename),"sha256":sha(payload),"bytes":len(payload)})
    if image_path.is_file():
        image_bytes=image_path.read_bytes()
        artifacts.append({"type":"image-output","path":str(image_path),"sha256":sha(image_bytes),"bytes":len(image_bytes)})
    response=(json.dumps({"gate":gates,"caller_success_counts":counts},sort_keys=True)+"\n").encode()
    (output/"response.json").write_bytes(response)
    finished=datetime.now(timezone.utc)
    started=datetime.fromisoformat(str(invocation["started_at"]).replace("Z","+00:00"))
    duration=max(1.0,(finished-started).total_seconds()*1000)
    fixture=sha((SCENARIO/"scripts/lm1_mixed_admission.contract.json").read_bytes())
    receipt={"schema_version":1,"operation_id":output.parent.name,"started_at":started.isoformat(),"finished_at":finished.isoformat(),
             "producer_digest":producer_digest,"contract_digest":contract_digest,"fixture_revision":fixture,
             "sample_count":1,"declared_warmups":0,"before":before,"after":candidate(api_url),
             "attempts":[{"index":0,"warmup":False,"status":"verified" if gates["passed"] else "failed",
                          "error":"" if gates["passed"] else "; ".join(k for k,v in gates.items() if k!="passed" and not v),
                          "operation_id":output.parent.name,"owner_duration_ms":duration,"wall_ms":duration,
                          "response_file":"response.json","response_sha256":sha(response),"artifacts":artifacts}],
             "errors":[] if gates["passed"] else ["LM1 mixed-admission gates failed"]}
    (output/"receipt.json").write_text(json.dumps(receipt,indent=2,sort_keys=True)+"\n")
    print(json.dumps({"gate":gates,"caller_success_counts":counts,"receipt":str(output/"receipt.json")},indent=2))
    return 0 if gates["passed"] else 1

if __name__ == "__main__":
    if len(sys.argv)!=3:
        raise SystemExit("usage: lm1_mixed_admission.py <framework-output-dir> <research-desk-api-url>")
    try: raise SystemExit(main(sys.argv[1],sys.argv[2]))
    except Exception as exc:
        print(f"LM1 mixed-admission producer refused: {exc}",file=sys.stderr)
        raise SystemExit(1)
