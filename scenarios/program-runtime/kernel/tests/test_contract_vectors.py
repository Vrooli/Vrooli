import json
from pathlib import Path

import pytest

from host.contracts import resolve_inputs


VECTORS = json.loads(
    (Path(__file__).parents[2] / "contracts" / "input-resolution-vectors.json").read_text()
)


@pytest.mark.parametrize("vector", VECTORS, ids=lambda vector: vector["name"])
def test_resolve_inputs_matches_shared_vectors(vector):
    if errors := vector.get("error_contains"):
        with pytest.raises((TypeError, ValueError)) as raised:
            resolve_inputs({"inputs": vector["inputs"]}, vector["provided"])
        assert any(error in str(raised.value) for error in errors)
        return

    assert resolve_inputs({"inputs": vector["inputs"]}, vector["provided"]) == vector["expected"]
