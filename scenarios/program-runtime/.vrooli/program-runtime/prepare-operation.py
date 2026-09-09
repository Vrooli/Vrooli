"""One bounded read of a selected program and its owning usage skill."""
import json
inputs = program.inputs()
envelope = {"program": "program-runtime.prepare-operation", "version": "1", "status": "failed",
            "phase": "report", "signals": {}, "errors": [], "evidence": []}
try:
    name = inputs["name"]
    if not isinstance(name, str) or len(name) > 256 or not all(c.isalnum() or c in "._-" for c in name):
        raise ValueError("Supply an exact program name")
    handle = program_runtime.library.get(name=name)
    rows = handle.head(1)
    entry = rows[0] if rows else handle.meta().get("program", {})
    if "program" in entry:
        entry = entry["program"]
    if entry.get("name") != name:
        raise ValueError("No matching program returned")
    envelope["signals"]["program"] = {k: entry.get(k) for k in (
        "name", "description", "contentDigest", "ownerSkill", "declaredInputs", "validationError", "calledBindingIds")}
    envelope["evidence"] = ["program:" + name + "@" + str(entry.get("contentDigest", ""))]
    envelope["signals"]["run_command"] = "program-runtime library run " + name + " --input '<key=value,...>'"
    envelope["status"] = "ok"
    owner = entry.get("ownerSkill")
    if owner:
        skill = prompt_manager.skill.read(identifiers=[owner], allow_missing=True, output="combined", rows="missing").meta()
        content = skill.get("combined", "")
        raw = content.encode("utf-8")
        envelope["signals"]["skill"] = {"id": owner, "content": raw[:24000].decode("utf-8", errors="ignore"),
            "hash": skill.get("combinedHash"), "truncated": len(raw) > 24000}
        if not content or len(raw) > 24000:
            envelope["status"] = "partial"
            envelope["errors"].append({"class": "skill_incomplete", "where": "skill",
                "detail": "Read the complete owner skill before execution", "command": "prompt-manager skill read " + owner})
    else:
        envelope["status"] = "partial"
        envelope["errors"].append({"class": "skill_incomplete", "where": "skill", "detail": "Program has no owning usage skill"})
    if entry.get("validationError"):
        envelope["status"] = "partial"
        envelope["errors"].append({"class": "invalid_contract", "where": "program", "detail": entry["validationError"]})
except Exception as exc:
    envelope["status"] = "partial" if envelope["signals"] else "unavailable"
    envelope["errors"].append({"class": "read_unavailable", "where": "collect", "detail": str(exc)[:240]})
print(json.dumps(envelope, ensure_ascii=False))
