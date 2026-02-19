#!/usr/bin/env bash
# sync-validation-results.sh
# Reads all module.json requirement validations and writes a phase-results
# JSON file that the completeness scorer can parse for test counts.
# Run after tests to ensure the scorer sees individual test-to-requirement mappings.

set -euo pipefail

export SCENARIO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export PHASE_DIR="${SCENARIO_ROOT}/coverage/phase-results"
export OUTPUT="${PHASE_DIR}/validation.json"

mkdir -p "$PHASE_DIR"

python3 -c "
import json, glob, os, datetime

scenario_root = os.environ['SCENARIO_ROOT']
reqs = []

for f in sorted(glob.glob(os.path.join(scenario_root, 'requirements', '*', 'module.json'))):
    with open(f) as fh:
        data = json.load(fh)
    for req in data.get('requirements', []):
        for v in req.get('validation', []):
            if v.get('type') == 'test':
                reqs.append({
                    'id': req['id'],
                    'title': req.get('title', ''),
                    'ref': v.get('ref', ''),
                    'status': 'passed' if v.get('status') == 'passing' else v.get('status', 'unknown')
                })

result = {
    'phase': 'validation',
    'scenario': 'web-console',
    'status': 'passed' if all(r['status'] == 'passed' for r in reqs) else 'failed',
    'summary': f'{sum(1 for r in reqs if r[\"status\"] == \"passed\")} passed, {sum(1 for r in reqs if r[\"status\"] == \"failed\")} failed',
    'requirements': reqs,
    'updated_at': datetime.datetime.now(datetime.timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')
}

with open(os.environ['OUTPUT'], 'w') as out:
    json.dump(result, out, indent=2)

print(f'Wrote {len(reqs)} validation entries to {os.environ[\"OUTPUT\"]}')
"
