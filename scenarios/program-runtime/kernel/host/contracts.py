"""Input admission for the immutable declaration supplied by the Go owner.

This mirrors Contract.ResolveInputs at the JSON boundary. Domain-specific nested
validation remains in the program. No binding is invoked during admission.
"""
import json
import math


def _json_copy(value):
    # The JSON encoder otherwise coerces object keys and tuples, which can
    # silently merge distinct keys or change a caller's supplied value.
    def validate(item, ancestors):
        if item is None or isinstance(item, (str, bool)):
            return
        if isinstance(item, (int, float)):
            if not _matches_type(item, "number"):
                raise ValueError("nonfinite number")
            return
        if not isinstance(item, (dict, list)):
            raise TypeError("unsupported JSON value")
        if id(item) in ancestors:
            raise ValueError("circular JSON value")
        ancestors.add(id(item))
        try:
            if isinstance(item, dict):
                if any(not isinstance(key, str) for key in item):
                    raise TypeError("JSON object keys must be strings")
                children = item.values()
            else:
                children = item
            for child in children:
                validate(child, ancestors)
        finally:
            ancestors.remove(id(item))

    validate(value, set())
    return json.loads(json.dumps(value, allow_nan=False))


def _matches_type(value, kind):
    if kind == "boolean":
        return isinstance(value, bool)
    if kind in ("number", "integer"):
        if isinstance(value, bool) or not isinstance(value, (int, float)):
            return False
        try:
            number = float(value)
        except OverflowError:
            return False
        return math.isfinite(number) and (kind == "number" or number.is_integer())
    types = {"string": str, "array": list, "object": dict}
    return kind in types and isinstance(value, types[kind])


def _json_equal(left, right):
    # Python equates True with 1, including inside containers. JSON does not.
    if isinstance(left, bool) or isinstance(right, bool):
        return type(left) is type(right) and left == right
    if isinstance(left, dict) and isinstance(right, dict):
        return left.keys() == right.keys() and all(_json_equal(left[k], right[k]) for k in left)
    if isinstance(left, list) and isinstance(right, list):
        return len(left) == len(right) and all(_json_equal(a, b) for a, b in zip(left, right))
    return left == right


def resolve_inputs(declaration, provided):
    specs = declaration.get("inputs", {})
    unknown = sorted(set(provided) - set(specs))
    if unknown:
        accepted = ", ".join(sorted(specs)) or "<none>"
        raise TypeError(f"unexpected keyword {unknown[0]!r}; accepted keywords: {accepted}")
    resolved = {}
    for name, spec in specs.items():
        if name in provided:
            value = provided[name]
        elif "default" in spec:
            value = spec["default"]
        elif spec.get("required", False):
            raise TypeError(f"missing required input {name!r}")
        else:
            continue
        if not _matches_type(value, spec["type"]):
            raise TypeError(f"input {name!r} must have type {spec['type']}")
        try:
            value = _json_copy(value)
        except (TypeError, ValueError, OverflowError, RecursionError) as exc:
            raise TypeError(f"input {name!r} must be a finite JSON value") from exc
        if spec.get("enum") and not any(_json_equal(value, candidate) for candidate in spec["enum"]):
            raise ValueError(f"input {name!r} is outside its declared enum")
        resolved[name] = value
    return resolved
