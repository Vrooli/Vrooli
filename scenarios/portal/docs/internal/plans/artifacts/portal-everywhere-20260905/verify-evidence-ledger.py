#!/usr/bin/env python3
"""Check ledger completeness, not producer authenticity or behavioral correctness."""
import json, pathlib, sys
base=pathlib.Path(__file__).resolve().parent
if len(sys.argv)!=2:
    print("usage: verify-evidence-ledger.py LEDGER.json",file=sys.stderr);sys.exit(2)
try:
    ledger=json.loads(pathlib.Path(sys.argv[1]).read_text())
    cases=json.loads((base/"acceptance-cases.json").read_text())
except (OSError,ValueError) as exc:
    print(str(exc),file=sys.stderr);sys.exit(2)
errors=[]
def fail(condition,message):
    if not condition: errors.append(message)
def check_rows(key,expected,fields,pathkey=None):
    rows=ledger.get(key,[])
    ids=[r.get("id") for r in rows]
    fail(len(ids)==len(set(ids)),key+": duplicate IDs")
    fail(set(ids)==set(expected),key+": required identity set differs")
    for row in rows:
        label=key+":"+str(row.get("id"))
        fail(row.get("status")=="passed",label+": not passed")
        for field in fields: fail(bool(row.get(field)),label+": missing "+field)
        if pathkey:
            for value in row.get(pathkey,[]):
                p=pathlib.Path(value)
                fail(p.is_absolute() and p.is_file(),label+": artifact is not an existing absolute file: "+value)
check_rows("cases",[c["id"] for c in cases],["owner","requirementIds","supportRows","producerRefs","assertionArtifacts","sourceFingerprint","observedAt"],"assertionArtifacts")
check_rows("platforms",["windows-x64","macos-arm64","linux-x64-x11","linux-x64-gnome-wayland","linux-x64-kde-wayland"],["producerRefs","artifactDigest","environment"])
check_rows("deliverables",[f"DEL-{i:02d}" for i in range(1,17)],["producerRefs","artifactPaths"],"artifactPaths")
for key in ["finalRegression"]:
    fail(ledger.get(key,{}).get("status")=="passed",key+": not passed")
    fail(bool(ledger.get(key,{}).get("producerRefs")),key+": missing producer reference")
for message in errors: print(message,file=sys.stderr)
print("ledger completeness: "+("failed" if errors else "passed; independently verify producer receipts"))
sys.exit(1 if errors else 0)
