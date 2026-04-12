#!/usr/bin/env python3

import json
import sys
import warnings
from pathlib import Path

import jsonschema


def main() -> int:
    warnings.filterwarnings(
        "ignore",
        category=DeprecationWarning,
        message=r".*RefResolver is deprecated.*",
    )
    repo_root = Path(__file__).resolve().parents[2]
    schema_dir = repo_root / ".vrooli" / "schemas"
    schema_path = schema_dir / "repo-contract.schema.json"
    contract_path = repo_root / ".vrooli" / "repo-contract.json"
    common_schema_path = schema_dir / "common.schema.json"

    with schema_path.open("r", encoding="utf-8") as fh:
        schema = json.load(fh)
    with contract_path.open("r", encoding="utf-8") as fh:
        contract = json.load(fh)
    with common_schema_path.open("r", encoding="utf-8") as fh:
        common_schema = json.load(fh)

    resolver = jsonschema.RefResolver(
        base_uri=schema_path.resolve().as_uri(),
        referrer=schema,
        store={
            schema_path.resolve().as_uri(): schema,
            common_schema_path.resolve().as_uri(): common_schema,
            "common.schema.json": common_schema,
        },
    )

    jsonschema.Draft7Validator.check_schema(schema)
    validator = jsonschema.Draft7Validator(schema, resolver=resolver)
    validator.validate(contract)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except jsonschema.ValidationError as exc:
        print(f"repo contract validation failed: {exc.message}", file=sys.stderr)
        raise SystemExit(1)
    except jsonschema.SchemaError as exc:
        print(f"repo contract schema is invalid: {exc.message}", file=sys.stderr)
        raise SystemExit(1)
