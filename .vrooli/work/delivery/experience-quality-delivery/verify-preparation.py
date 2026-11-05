"""Read-only integrity check for this disabled receiving handoff, not a product gate."""

from pathlib import Path
import hashlib
import json
import re
import xml.etree.ElementTree as ET
import zipfile


HOME = Path(__file__).resolve().parent
REPO = HOME.parents[2]
errors = []


def require(condition, message):
    if not condition:
        errors.append(message)


required = [
    "README.md", "GOAL.md", "QUEUE.md", "COMMISSION.md", "FEEDBACK.md",
    "WORKAROUNDS.md", "INSTALLS.md", "TARGETS.md", "RESEARCH.md", "GAPS.md",
    "DECISIONS.md", "VALIDATION.md", "SETUP.md", "sources.json",
    "native/operating-contract.json", "native/create-team.request.json",
    "native/AGENT.md", "native/RESPONSIBILITIES.md", "native/HEARTBEAT.md",
    "native/leader-binding.proposed.json", "native/preparation-receipt.json",
]
for relative in required:
    require((HOME / relative).is_file(), f"Missing handoff file: {relative}")

goal = (HOME / "GOAL.md").read_text()
require(len(goal) <= 2048, f"GOAL exceeds harness cap: {len(goal)} characters")
resume = sum((HOME / name).stat().st_size for name in (
    "GOAL.md", "QUEUE.md", "FEEDBACK.md", "WORKAROUNDS.md"
))
require(resume < 30000, f"Resume set exceeds 30 KB: {resume}")
queue = (HOME / "QUEUE.md").read_text()
for heading in ("Forecast", "Handoff", "Waiting on others", "Needs operator"):
    require(queue.count(f"## {heading}\n") == 1, f"Expected one queue {heading}")
require(not list((HOME / "epochs").glob("E*.md")), "Preparation contains an admitted epoch")

sources = json.loads((HOME / "sources.json").read_text())
for item in sources["files"]:
    path = REPO / item["path"]
    if not path.is_file():
        require(False, f"Missing received/research source: {item['path']}")
        continue
    content = path.read_bytes()
    require(len(content) == item["bytes"], f"Source byte drift: {item['path']}")
    require(hashlib.sha256(content).hexdigest() == item["sha256"],
            f"Source hash drift: {item['path']}")

archive = REPO / sources["archive"]["path"]
require(hashlib.sha256(archive.read_bytes()).hexdigest() == sources["archive"]["sha256"],
        "Original dossier ZIP hash mismatch")
expanded = HOME / "sources/design-product-dossier-1.3"
declared = json.loads((expanded / "manifest.json").read_text())
with zipfile.ZipFile(archive) as bundle:
    for item in declared["files"]:
        content = (expanded / item["path"]).read_bytes()
        require(bundle.read(item["path"]) == content, f"ZIP/expanded mismatch: {item['path']}")
        require(len(content) == item["bytes"], f"Dossier byte mismatch: {item['path']}")
        require(hashlib.sha256(content).hexdigest() == item["sha256"],
                f"Dossier hash mismatch: {item['path']}")
ET.parse(expanded / "assets/icon.svg")

target = (REPO / "docs/architecture/experience-quality.md").read_text()
for index in range(1, 19):
    require(f"| EQ-{index:02} |" in target, f"Missing outcome EQ-{index:02}")
validation = (HOME / "VALIDATION.md").read_text()
for index in range(1, 15):
    require(f"### V{index:02} " in validation, f"Missing acceptance V{index:02}")

# Check this handoff's actual Markdown links; snapshots keep their source scope.
docs = list(HOME.glob("*.md")) + [REPO / "docs/architecture/experience-quality.md"]
for document in docs:
    for destination in re.findall(r"\]\(([^)]+)\)", document.read_text()):
        if destination.startswith(("https://", "http://", "#", "mailto:")):
            continue
        local = destination.split("#", 1)[0]
        require((document.parent / local).exists(),
                f"Broken local link in {document.name}: {destination}")

request = json.loads((HOME / "native/create-team.request.json").read_text())["team"]
require(request["enabled"] is False, "Prepared team request permits activation")
require(request["purpose"] == "delivery" and request["lifetime"] == "finite",
        "Prepared native team is not finite delivery")
require(request["operatingContract"] == json.loads(
    (HOME / "native/operating-contract.json").read_text()), "Contract/request mismatch")
receipt = json.loads((HOME / "native/preparation-receipt.json").read_text())
require(receipt["team"]["enabled"] is False, "Receipt shows team enabled")
require(receipt["heartbeat"]["enabled"] is False and
        receipt["heartbeat"]["scheduled"] is False, "Receipt shows active heartbeat")
require(receipt["finite_binding"]["state"] == "unapplied_proposal",
        "Historical finite binding incorrectly reported as applied")

if errors:
    print("PREPARATION INTEGRITY FAILED")
    for error in errors:
        print(f"- {error}")
    raise SystemExit(1)
print(f"Preparation integrity passed: {len(sources['files'])} sources; "
      f"{len(declared['files'])} dossier entries; GOAL {len(goal)} characters; "
      f"resume {resume}/30000 bytes; 18 outcomes; 14 planned acceptance journeys.")
print("This check does not prove native execution, UI behavior, or release readiness.")
