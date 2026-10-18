"""Offline song resource: bounded requests, isolated workers, owned model retention.

Only the supervisor process serves HTTP. Heavy libraries are imported by one
short-lived worker; its exit releases every model allocation, including on error.
"""
from __future__ import annotations

import contextlib
import fcntl
import hashlib
import json
import math
import os
from pathlib import Path
import shutil
import signal
import socket
import subprocess
import sys
import threading
import time
import uuid
import wave


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), allow_nan=False).encode()


def digest(path):
    with Path(path).open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def reject_nonfinite(_):
    raise ValueError("nonfinite usage timestamp")


def write_json(path, value):
    path = Path(path)
    temporary = path.with_name(path.name + "." + uuid.uuid4().hex + ".tmp")
    try:
        with temporary.open("xb") as stream:
            stream.write(canonical(value) + b"\n")
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, path)
        directory = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directory)
        finally:
            os.close(directory)
    finally:
        temporary.unlink(missing_ok=True)


class Refused(Exception):
    def __init__(self, message, status=400):
        super().__init__(message)
        self.status = status


class Runtime:
    def __init__(self, root, state, artifact, lock_path, *, popen=subprocess.Popen, now=time.time):
        self.root, self.state, self.artifact = map(Path, (root, state, artifact))
        self.lock_path = Path(lock_path)
        self.catalog_path = self.artifact / "source/profiles.json"
        self.catalog = json.loads(self.catalog_path.read_text())
        if self.catalog.get("schema_version") != 1 or not self.catalog.get("profiles"):
            raise ValueError("unsupported or empty song profile catalog")
        self.profiles = self.catalog["profiles"]
        self.catalog_sha = digest(self.catalog_path)
        self.artifact_sha = os.environ.get("SONGGEN_COMPOSED_SHA256", "")
        self.popen, self.now = popen, now
        self.guard = threading.Lock()
        self.active = None
        self.process = None
        self.stopping = threading.Event()
        self.outputs = self.root / "outputs"
        self.outputs.mkdir(parents=True, exist_ok=True)
        self.state.mkdir(parents=True, exist_ok=True)
        self.usage_path = self.state / "model-usage.json"
        self.observe_installations()

    @contextlib.contextmanager
    def data_lock(self):
        # The control plane supplies its canonical resource-data lock path.
        # Never create a second lock spelling or guess the user's runtime home.
        self.lock_path.parent.mkdir(parents=True, exist_ok=True)
        with self.lock_path.open("a+b") as stream:
            try:
                fcntl.flock(stream, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError as exc:
                raise Refused("model data is in use or being acquired", 409) from exc
            try:
                yield
            finally:
                fcntl.flock(stream, fcntl.LOCK_UN)

    def model_root(self, profile):
        root = self.root / profile["model_root"]
        if root.parent != self.root / "models" or root.is_symlink() or root.parent.is_symlink():
            raise Refused("invalid model bundle root", 409)
        return root

    def available(self, profile):
        root = self.model_root(profile)
        for entry in profile["files"]:
            path = self.root / entry["path"]
            if path.is_symlink() or not path.is_file() or path.stat().st_size != entry["bytes"]:
                return False
            if not path.resolve().is_relative_to(root.resolve()):
                return False
        return True

    def usage(self):
        if not self.usage_path.exists():
            return {}
        try:
            value = json.loads(self.usage_path.read_text(), parse_constant=reject_nonfinite)
        except (ValueError, OSError) as exc:
            raise Refused("model usage state is corrupt; cleanup refused", 409) from exc
        if not isinstance(value, dict):
            raise Refused("model usage state is corrupt; cleanup refused", 409)
        return value

    def installation_fingerprint(self, profile):
        if not self.available(profile):
            return None
        files = []
        for entry in profile["files"]:
            stat = (self.root / entry["path"]).stat()
            files.append([entry["path"], stat.st_size, stat.st_mtime_ns, stat.st_ino])
        return hashlib.sha256(canonical(files)).hexdigest()

    def observe_installations(self):
        # First observation starts a fresh retention window, even if a profile
        # has never generated. Reacquired bundles receive a fresh window too.
        with self.data_lock():
            usage = self.usage()
            changed = False
            for key, profile in self.profiles.items():
                fingerprint = self.installation_fingerprint(profile)
                if fingerprint is None:
                    continue
                previous = usage.get(key, {})
                if not isinstance(previous, dict):
                    raise Refused("model usage entry is corrupt", 409)
                if previous.get("installation_fingerprint") != fingerprint or previous.get("revision") != profile["revision"]:
                    usage[key] = {"revision": profile["revision"], "installation_fingerprint": fingerprint,
                                  "installed_observed_at": self.now(), "pinned": previous.get("pinned", False)}
                    changed = True
            if changed:
                write_json(self.usage_path, usage)

    def health(self):
        profiles = {key: {"available": self.available(p), "model_id": p["model_id"],
                          "revision": p["revision"], "sung_lyrics": p["sung_lyrics"],
                          "voice_reference": p["voice_reference"]} for key, p in self.profiles.items()}
        return {"status": "ready" if any(p["available"] for p in profiles.values()) else "unavailable",
                "protocol": "song-v1", "catalog_sha256": self.catalog_sha, "artifact_sha256": self.artifact_sha,
                "active_operation": self.active, "model_loaded": None if self.process is not None else False,
                "worker_running": self.process is not None, "workload_active": self.active is not None,
                "profiles": profiles}

    def validate(self, request):
        fields = {"profile", "prompt", "lyrics", "duration", "seed", "steps", "guidance", "expected_revision", "timeout_seconds"}
        if not isinstance(request, dict) or set(request) != fields:
            raise Refused("request requires exactly: " + ", ".join(sorted(fields)))
        p = self.profiles.get(request["profile"])
        if p is None or request["expected_revision"] != p["revision"]:
            raise Refused("unknown profile or mismatched model revision")
        for name, limit in (("prompt", 12000), ("lyrics", 16000)):
            value = request[name]
            if not isinstance(value, str) or not value.strip() or len(value) > limit:
                raise Refused(f"{name} must be nonblank and at most {limit} characters")
        for name in ("duration", "guidance"):
            value = request[name]
            if type(value) not in (int, float) or not math.isfinite(value):
                raise Refused(f"{name} must be finite")
        if not 1 <= request["duration"] <= p["max_duration"] or not p["min_guidance"] <= request["guidance"] <= p["max_guidance"]:
            raise Refused("duration or guidance outside profile limits")
        for name in ("seed", "steps", "timeout_seconds"):
            if type(request[name]) is not int:
                raise Refused(f"{name} must be an integer")
        if not 0 <= request["seed"] < 2**32 or not 1 <= request["timeout_seconds"] <= p["max_execution_seconds"]:
            raise Refused("seed or timeout outside profile limits")
        if not p["min_steps"] <= request["steps"] <= p["max_steps"]:
            raise Refused("steps outside supported profile controls")
        return p

    def terminate(self):
        proc = self.process
        if proc is None or proc.poll() is not None:
            return
        try:
            os.killpg(proc.pid, signal.SIGTERM)
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            os.killpg(proc.pid, signal.SIGKILL)
            proc.wait(timeout=5)
        except ProcessLookupError:
            proc.wait(timeout=5)

    def generate(self, request, disconnected=lambda: False, *, raw=None):
        p = self.validate(request)
        if not self.guard.acquire(blocking=False):
            raise Refused("one song generation or model mutation is already active", 409)
        stage = None
        try:
            if self.stopping.is_set():
                raise Refused("resource is stopping", 503)
            with self.data_lock():
                if not self.available(p):
                    raise Refused(f"model files unavailable; use vrooli resource acquisition data songgen --group {request['profile']}", 503)
                operation = uuid.uuid4().hex
                stage = self.outputs / (".stage-" + operation)
                stage.mkdir(mode=0o700)
                (stage / ".active").touch()
                self.active = operation
                usage = self.usage()
                prior = usage.get(request["profile"], {})
                usage[request["profile"]] = {**prior, "last_used": self.now(), "pinned": prior.get("pinned", False), "revision": p["revision"],
                                             "installation_fingerprint": self.installation_fingerprint(p)}
                usage[request["profile"]].pop("retiring_catalog", None)
                write_json(self.usage_path, usage)
                # New work is the bounded sweep trigger. The selected bundle was
                # just touched; only expired, unpinned peers can be reclaimed.
                try:
                    self.apply_cleanup(self.model_preview(usage), usage)
                except OSError as exc:
                    print(f"expired peer cache cleanup deferred: {exc}", file=sys.stderr, flush=True)
                job = {"request": request, "profile": p, "catalog_sha256": self.catalog_sha,
                       "data_root": str(self.root), "artifact_root": str(self.artifact), "parent_pid": os.getpid()}
                write_json(stage / "job.json", job)
                env = {k: v for k, v in os.environ.items() if k not in {"PYTHONPATH", "PYTHONHOME", "HF_TOKEN", "HUGGING_FACE_HUB_TOKEN"}}
                env.update({"HF_HUB_OFFLINE": "1", "TRANSFORMERS_OFFLINE": "1", "HF_HUB_DISABLE_TELEMETRY": "1",
                            "PYTHONDONTWRITEBYTECODE": "1", "HF_HOME": str(self.root / "cache/huggingface"),
                            "XDG_CACHE_HOME": str(self.root / "cache"), "TORCH_HOME": str(self.root / "cache/torch")})
                with (stage / "worker.log").open("wb") as log:
                    self.process = self.popen([str(self.artifact / "runtime/bin/python"), "-I", "-B",
                                              str(self.artifact / "source/worker.py"), str(stage)],
                                             stdin=subprocess.DEVNULL, stdout=log, stderr=subprocess.STDOUT,
                                             env=env, start_new_session=True)
                    deadline = time.monotonic() + request["timeout_seconds"]
                    while self.process.poll() is None:
                        if self.stopping.is_set() or disconnected() or time.monotonic() >= deadline:
                            self.terminate()
                            raise Refused("generation cancelled or timed out; worker exited without publication", 408)
                        time.sleep(0.1)
                    if self.process.returncode != 0:
                        raise Refused(f"song worker failed (exit {self.process.returncode}); inspect resource logs", 502)
                audio = stage / "audio.wav"
                result = json.loads((stage / "result.json").read_text())
                if result["seed"] != request["seed"] or result["model_id"] != p["model_id"] or result["revision"] != p["revision"]:
                    raise Refused("worker returned mismatched model or seed", 502)
                with wave.open(str(audio)) as wav:
                    frames, rate, channels = wav.getnframes(), wav.getframerate(), wav.getnchannels()
                    if wav.getsampwidth() not in (2, 3) or channels not in (1, 2) or not 16000 <= rate <= 96000 or not 0 < frames / rate <= request["duration"] + 1:
                        raise Refused("worker returned invalid PCM audio", 502)
                receipt = {"schema_version": 1, "status": "completed", "operation_id": operation,
                           "request_sha256": hashlib.sha256(raw if raw is not None else canonical(request)).hexdigest(), "effective_request": request,
                           "catalog_sha256": self.catalog_sha, "artifact_sha256": self.artifact_sha, "model": {"id": p["model_id"], "revision": p["revision"],
                           "source_revision": p["source_revision"], "license": p["license"]}, "worker": result,
                           "output": {"sha256": digest(audio), "bytes": audio.stat().st_size, "frames": frames,
                           "sample_rate": rate, "channels": channels, "duration_seconds": frames / rate}}
                write_json(stage / "receipt.json", receipt)
                (stage / ".active").unlink()
                # UUID names are resource-created, never supplied by a caller.
                final = self.outputs / operation
                stage.rename(final)
                stage = None
                return receipt
        except BaseException as exc:
            details = {"status": "failed", "profile": request["profile"], "operation_id": self.active,
                       "error": str(exc)[:4096], "time": self.now()}
            if stage is not None and (stage / "worker.log").exists():
                with (stage / "worker.log").open("rb") as log:
                    log.seek(max(0, (stage / "worker.log").stat().st_size - 8192))
                    details["worker_log_tail"] = log.read(8192).decode(errors="replace")
            try:
                write_json(self.state / "last-failure.json", details)
            except OSError as record_error:
                print("failed to persist resource failure: " + str(record_error), file=sys.stderr, flush=True)
            print(json.dumps(details, allow_nan=False), file=sys.stderr, flush=True)
            raise
        finally:
            self.terminate()
            self.process, self.active = None, None
            if stage is not None:
                shutil.rmtree(stage, ignore_errors=True)
            self.guard.release()

    def model_preview(self, usage):
        items = []
        for key, p in sorted(self.profiles.items()):
            entry = usage.get(key)
            root = self.model_root(p)
            if not root.exists():
                continue
            timestamps = [entry[name] for name in ("last_used", "installed_observed_at") if isinstance(entry, dict) and type(entry.get(name)) in (int, float) and math.isfinite(entry[name])]
            known = isinstance(entry, dict) and entry.get("revision") == p["revision"] and bool(timestamps) and (bool(entry.get("installation_fingerprint")) and entry.get("installation_fingerprint") == self.installation_fingerprint(p) or (entry.get("retiring_catalog") == self.catalog_sha and not self.available(p)))
            pinned = not known or entry.get("pinned", True) is not False
            eligible = known and not pinned and self.now() - max(timestamps) >= p["retention_days"] * 86400
            expected = {str(self.root / f["path"]) for f in p["files"]}
            metadata = []
            for path in sorted(root.rglob("*")):
                if path.is_symlink():
                    eligible = False
                    continue
                if not path.is_file() and not path.is_dir():
                    eligible = False
                if path.is_file():
                    if str(path) not in expected and not (str(path).endswith(".sha256") and str(path)[:-7] in expected):
                        eligible = False
                    stat = path.stat()
                    metadata.append([str(path.relative_to(root)), stat.st_size, stat.st_mtime_ns, stat.st_ino])
            items.append({"profile": key, "eligible": bool(eligible), "pinned_or_unknown": pinned,
                          "last_used": entry.get("last_used") if known else None,
                          "installed_observed_at": entry.get("installed_observed_at") if known else None, "bytes": sum(m[1] for m in metadata),
                          "snapshot_sha256": hashlib.sha256(canonical([key, entry, metadata])).hexdigest()})
        return {"catalog_sha256": self.catalog_sha, "items": items}

    def apply_cleanup(self, preview, usage):
        """Caller holds both mutation and acquisition fences.

        Persist retirement intent before deletion. A partial filesystem failure
        remains a declared bundle eligible for a fresh, reviewed retry rather
        than becoming an untracked hidden directory or an immortal partial tree.
        Unknown files, symlinks and pins still veto each subsequent preview.
        """
        removed = []
        for item in preview["items"]:
            if not item["eligible"]:
                continue
            key = item["profile"]
            usage[key]["retiring_catalog"] = self.catalog_sha
            write_json(self.usage_path, usage)
            shutil.rmtree(self.model_root(self.profiles[key]))
            usage[key].pop("retiring_catalog", None)
            write_json(self.usage_path, usage)
            removed.append(key)
        return {"removed_profiles": removed, "preview": preview}

    def models(self, request):
        if not isinstance(request, dict) or set(request) - {"dry_run", "preview", "pin"}:
            raise Refused("invalid model mutation request")
        if "dry_run" in request and type(request["dry_run"]) is not bool:
            raise Refused("dry_run must be a boolean")
        if "pin" in request and ("preview" in request or request.get("dry_run") is False):
            raise Refused("pin and cleanup must be separate operations")
        if not self.guard.acquire(blocking=False):
            raise Refused("model mutation refused while generation is active", 409)
        try:
            with self.data_lock():
                usage = self.usage()
                if "pin" in request:
                    pins = request["pin"]
                    if not isinstance(pins, dict) or any(key not in self.profiles or type(value) is not bool for key, value in pins.items()):
                        raise Refused("pin requires declared profile identifiers and booleans")
                    for key, value in pins.items():
                        entry = usage.setdefault(key, {"revision": self.profiles[key]["revision"]})
                        entry["pinned"] = value
                    write_json(self.usage_path, usage)
                preview = self.model_preview(usage)
                if request.get("dry_run", True) is not False:
                    return preview
                if request.get("preview") != preview:
                    raise Refused("model cleanup preview changed; inspect a fresh preview", 409)
                return self.apply_cleanup(preview, usage)
        finally:
            self.guard.release()


def connection_closed(connection):
    try:
        return connection.recv(1, socket.MSG_PEEK | socket.MSG_DONTWAIT) == b""
    except (BlockingIOError, InterruptedError):
        return False
    except OSError:
        return True
