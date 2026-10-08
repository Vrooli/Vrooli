"""Fleet contract: a learning program must hand its feedback reference back to the caller.

A correction is usually discovered later than the run that earned it — the output looked
fine, and only a downstream consumer or the operator finds out it was not. `learn.feedback`
can record that without repeating any domain work, but only if the caller still holds the
reference. A program that learns and then drops its reference makes delayed feedback
impossible, which silently removes the half of the loop that carries real-world usefulness.

These tests walk every declared program in the repository, so a new learning program cannot
be added without a path back to it.
"""
import json
import re
from pathlib import Path

ROOT = Path(__file__).parents[4]
SIGNAL = "learning"
# The kernel allocates a durable reference for these verbs; learn.task alone still produces a
# gradeable attempt identity, so every learning program has something a caller can name later.
LEARN_VERB = re.compile(r"\blearn\.[a-z_]+\s*\(")
# Either a dict literal ("learning": {...}) or a subscript assignment (…["learning"] = {…}).
EMITS_SIGNAL = re.compile(r"""["']learning["']\s*(?::|\]\s*=)""")


def declared_programs():
    for contract_path in sorted(ROOT.glob("scenarios/*/.vrooli/program-runtime/*.json")):
        source_path = contract_path.with_suffix(".py")
        if not source_path.exists():
            continue
        try:
            contract = json.loads(contract_path.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            continue
        if not isinstance(contract, dict) or "name" not in contract:
            continue
        yield contract, contract_path, source_path


def learning_programs():
    for contract, contract_path, source_path in declared_programs():
        source = source_path.read_text(encoding="utf-8")
        declares = [verb for verb in contract.get("verbs", []) if str(verb).startswith("learn.")]
        if declares or LEARN_VERB.search(source):
            yield contract, contract_path, source_path, source


def test_every_learning_program_declares_a_learning_output_signal():
    """[REQ:LV-12] The caller can find the feedback reference in the documented output shape."""
    missing = [
        contract["name"]
        for contract, _path, _source_path, _source in learning_programs()
        if SIGNAL not in ((contract.get("outputs") or {}).get("signals") or {})
    ]
    assert not missing, (
        "programs use learning verbs but never declare a 'learning' output signal, so a later "
        "learn.feedback has no reference to name: " + ", ".join(sorted(missing))
    )


def test_every_learning_program_emits_the_learning_block_in_its_envelope():
    """[REQ:LV-12] The declared signal is actually populated, not only documented."""
    silent = [
        contract["name"]
        for contract, _path, _source_path, source in learning_programs()
        if not EMITS_SIGNAL.search(source)
    ]
    assert not silent, (
        "programs declare learning verbs but never write a 'learning' key into their result, so "
        "the reference never reaches the caller: " + ", ".join(sorted(silent))
    )


def test_the_learning_signal_documents_how_to_use_the_reference():
    """A caller that cannot tell what the field is for will not keep it."""
    unclear = []
    for contract, _path, _source_path, _source in learning_programs():
        description = ((contract.get("outputs") or {}).get("signals") or {}).get(SIGNAL, "")
        if "feedback_ref" not in str(description):
            unclear.append(contract["name"])
    assert not unclear, (
        "the 'learning' signal must name feedback_ref so a caller knows what to preserve for a "
        "later correction: " + ", ".join(sorted(unclear))
    )
