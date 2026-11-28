"""A binding-free program runs outside Program Runtime with the kernel's helper."""
import json
import subprocess
import sys
from pathlib import Path

RUNNER = Path(__file__).resolve().parents[1] / "host" / "standalone.py"


def program(tmp_path, source, inputs=None, bindings=()):
    contract = tmp_path / "pulse.json"
    contract.write_text(json.dumps({"name": "research/pulse", "helpers": {"program": "2"}, "bindings": list(bindings),
                                    "inputs": inputs or {"limit": {"type": "integer", "default": 2}},
                                    "outputs": {"status": {"enum": ["ok", "failed"]}}}))
    (tmp_path / "pulse.py").write_text(source)
    return contract


def run(contract, *args):
    done = subprocess.run([sys.executable, "-I", str(RUNNER), str(contract), *args], capture_output=True, text=True, check=False)
    return done.returncode, done.stdout, done.stderr


OK = """envelope = program.envelope("research/pulse", "1")
envelope["signals"] = {"limit": program.inputs()["limit"]}
envelope["status"] = "ok"
program.report()
"""


def test_runs_with_declared_defaults_and_typed_inputs(tmp_path):
    contract = program(tmp_path, OK)
    code, out, _ = run(contract)
    assert code == 0 and json.loads(out)["signals"] == {"limit": 2}
    code, out, _ = run(contract, "limit=7")
    assert code == 0 and json.loads(out)["inputs"] == {"limit": 7}
    code, out, _ = run(contract, "--inputs-json", '{"limit": 9}')
    assert code == 0 and json.loads(out)["signals"] == {"limit": 9}


def test_rejects_inputs_the_kernel_would_reject(tmp_path):
    contract = program(tmp_path, OK, inputs={"limit": {"type": "integer", "required": True}})
    for args, message in ((["limit=1.5"], "must have type integer"), ([], "missing required input"), (["limit=1", "other=2"], "unknown input")):
        code, _, err = run(contract, *args)
        assert code == 2 and message in err


def test_refuses_programs_that_need_the_kernel(tmp_path):
    code, out, _ = run(program(tmp_path, "rows = gather(lambda: 1)\nprogram.report()\n"))
    envelope = json.loads(out)
    assert code == 3 and envelope["status"] == "unavailable"
    assert "needs Program Runtime for gather" in envelope["errors"][0]["detail"]
    code, out, _ = run(program(tmp_path, OK, bindings=[{"id": "agent-manager/run/report", "effect": "read"}]))
    assert code == 3 and "agent-manager/run/report" in out


def test_a_name_the_program_defines_itself_is_not_a_kernel_verb(tmp_path):
    source = "def validate():\n    return 'report'\n" + OK
    code, out, _ = run(program(tmp_path, source))
    assert code == 0 and json.loads(out)["status"] == "ok"


def test_a_failed_envelope_exits_nonzero(tmp_path):
    source = 'program.envelope("research/pulse", "1")\nprogram.run({"validate": lambda: program.fail("failed", "invalid_input", "bad", "validate"), "report": lambda: program.report()})\n'
    code, out, _ = run(program(tmp_path, source))
    assert code == 1 and json.loads(out)["status"] == "failed"
