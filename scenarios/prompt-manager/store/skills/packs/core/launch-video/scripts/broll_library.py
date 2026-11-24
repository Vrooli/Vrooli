#!/usr/bin/env python3
"""Hash-bound B-roll catalog and deterministic selection for launch-video plans.

This tool audits local media and ranks candidate plates. It does not grant rights,
approve taste, or claim that a visual proves a product capability.
"""
import argparse
import hashlib
import json
import subprocess
from pathlib import Path


def sha256(path):
    h = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def probe(path):
    proc = subprocess.run(['ffprobe', '-v', 'error', '-select_streams', 'v:0',
                           '-show_entries', 'stream=width,height,codec_name',
                           '-of', 'json', str(path)], capture_output=True, text=True)
    if proc.returncode:
        raise ValueError(f'ffprobe could not read {path}: {proc.stderr[-500:]}')
    streams = json.loads(proc.stdout).get('streams', [])
    if not streams:
        raise ValueError(f'no video/image stream: {path}')
    return {k: streams[0][k] for k in ('width', 'height', 'codec_name') if k in streams[0]}


def audit(manifest_path):
    manifest_path = Path(manifest_path).resolve()
    data = json.loads(manifest_path.read_text())
    errors, receipts = [], []
    ids = set()
    for item in data.get('assets', []):
        label = item.get('id', '<missing-id>')
        if not item.get('id') or label in ids:
            errors.append(f'{label}: missing or duplicate id')
        ids.add(label)
        for field in ('product', 'roles', 'claims', 'states'):
            if not isinstance(item.get(field), list) or not all(isinstance(v, str) and v for v in item.get(field, [])):
                errors.append(f'{label}: {field} must be a list of non-empty strings')
        if not isinstance(item.get('parameters'), dict):
            errors.append(f'{label}: parameters must be an object')
        if not isinstance(item.get('selection_eligible'), bool):
            errors.append(f'{label}: selection_eligible must be an explicit boolean')
        if not isinstance(item.get('dimensions'), list) or len(item.get('dimensions', [])) != 2 or not all(isinstance(v, int) and v > 0 for v in item.get('dimensions', [])):
            errors.append(f'{label}: dimensions must be two positive integers')
        path = Path(item.get('path', ''))
        if not path.is_absolute():
            path = manifest_path.parent / path
        if not path.is_file():
            errors.append(f'{label}: source missing: {path}')
            continue
        actual = sha256(path)
        if item.get('sha256') != actual:
            errors.append(f'{label}: source SHA-256 mismatch (expected {item.get("sha256")}, got {actual})')
        try:
            dimensions = probe(path)
        except (ValueError, KeyError) as exc:
            errors.append(f'{label}: {exc}')
            continue
        if item.get('dimensions') != [dimensions.get('width'), dimensions.get('height')]:
            errors.append(f'{label}: dimensions mismatch (expected {item.get("dimensions")}, got {[dimensions.get("width"), dimensions.get("height")]})')
        missing = [k for k in ('origin', 'rights_status', 'proof_scope', 'product', 'roles', 'claims', 'states', 'parameters') if k not in item]
        if missing:
            errors.append(f'{label}: missing required provenance/selection fields: {", ".join(missing)}')
        if item.get('rights_status') not in ('internal-generated-terms-unverified', 'owned-capture', 'licensed-with-receipt'):
            errors.append(f'{label}: invalid rights_status; do not promote without source evidence')
        if item.get('rights_status') == 'licensed-with-receipt' and not item.get('rights_receipt'):
            errors.append(f'{label}: licensed-with-receipt requires a rights_receipt reference')
        if item.get('rights_status') == 'owned-capture' and not item.get('capture_receipt'):
            errors.append(f'{label}: owned-capture requires a capture_receipt reference')
        receipt_field = ('rights_receipt' if item.get('rights_status') == 'licensed-with-receipt'
                         else 'capture_receipt' if item.get('rights_status') == 'owned-capture' else None)
        if receipt_field and item.get(receipt_field):
            receipt = item[receipt_field]
            if not isinstance(receipt, dict) or not receipt.get('path') or not receipt.get('sha256'):
                errors.append(f'{label}: {receipt_field} must contain path and sha256')
            else:
                receipt_path = Path(receipt['path'])
                if not receipt_path.is_absolute():
                    receipt_path = manifest_path.parent / receipt_path
                if not receipt_path.is_file():
                    errors.append(f'{label}: {receipt_field} missing: {receipt_path}')
                elif sha256(receipt_path) != receipt['sha256']:
                    errors.append(f'{label}: {receipt_field} SHA-256 mismatch: {receipt_path}')
        for index, receipt in enumerate(item.get('source_receipts', [])):
            if not isinstance(receipt, dict) or not receipt.get('path') or not receipt.get('sha256'):
                errors.append(f'{label}: source_receipts[{index}] must contain path and sha256')
                continue
            receipt_path = Path(receipt['path'])
            if not receipt_path.is_absolute():
                receipt_path = manifest_path.parent / receipt_path
            if not receipt_path.is_file():
                errors.append(f'{label}: source_receipts[{index}] missing: {receipt_path}')
            elif sha256(receipt_path) != receipt['sha256']:
                errors.append(f'{label}: source_receipts[{index}] SHA-256 mismatch: {receipt_path}')
        if item.get('rights_status') == 'internal-generated-terms-unverified' and item.get('selection_eligible'):
            errors.append(f'{label}: unresolved generated terms cannot be marked selection eligible')
        if 'concept' in str(item.get('proof_scope', '')).lower() and item.get('claims'):
            errors.append(f'{label}: conceptual material cannot claim product proof')
        receipts.append({'id': label, 'path': str(path), 'sha256': actual,
                         'dimensions': [dimensions.get('width'), dimensions.get('height')],
                         'origin': item.get('origin'), 'rights_status': item.get('rights_status'),
                         'capture_receipt': item.get('capture_receipt'),
                         'rights_receipt': item.get('rights_receipt'),
                         'source_receipts': item.get('source_receipts', []),
                         'proof_scope': item.get('proof_scope'), 'product': item.get('product'),
                         'roles': item.get('roles', []), 'claims': item.get('claims', []), 'states': item.get('states', []),
                         'parameters': item.get('parameters'),
                         'eligible_for_selection': bool(item.get('selection_eligible', False))})
    return data, receipts, errors


def selection_blockers(item):
    """Explain recorded selection holds without inferring taste or source rights."""
    blockers = []
    if not item.get('selection_eligible', False):
        blockers.append('catalog selection_eligible is false; evidence-owner review is pending')
    if item.get('rights_status') == 'internal-generated-terms-unverified':
        blockers.append('generated-media terms are unverified')
    if 'concept' in str(item.get('proof_scope', '')).lower():
        blockers.append('concept-only material cannot substantiate product claims')
    if any(any(marker in str(state).lower() for marker in ('fixture', 'old-'))
           for state in item.get('states', [])):
        blockers.append('catalog state identifies fixture or old-build material; current product relevance is unverified')
    if not item.get('claims'):
        blockers.append('no product claim is linked; use only as non-claim B-roll')
    return blockers


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('manifest', help='JSON catalog; paths may be relative to its directory')
    parser.add_argument('--product', help='candidate product filter (aquila or vega)')
    parser.add_argument('--role', help='candidate role such as opener, breath, bridge, payoff')
    parser.add_argument('--claim', help='exact claim id required for candidate matching')
    parser.add_argument('--state', help='exact product-state tag, for example retained-ui-fixture')
    parser.add_argument('--out', help='write audit/selection receipt JSON')
    args = parser.parse_args()
    data, receipts, errors = audit(args.manifest)
    ranked = []
    for item in data.get('assets', []):
        if args.product and args.product not in item.get('product', []):
            continue
        if args.state and args.state not in item.get('states', []):
            continue
        if args.role and args.role not in item.get('roles', []):
            continue
        if args.claim and args.claim not in item.get('claims', []):
            continue
        # Candidate-only unless an evidence owner later confirms eligibility.
        rank = (int(bool(args.state and args.state in item.get('states', []))) +
                int(bool(args.claim and args.claim in item.get('claims', []))) +
                int(bool(args.role and args.role in item.get('roles', []))) +
                int(bool(args.product and args.product in item.get('product', []))))
        blockers = selection_blockers(item)
        ranked.append({'id': item.get('id'), 'rank': rank,
                       'selection_eligible': bool(item.get('selection_eligible', False)),
                       'blockers': blockers,
                       'reason': 'ranked metadata match; candidate only, not taste approval'})
    ranked.sort(key=lambda row: (-row['rank'], row['id'] or ''))
    result = {'schema': 1, 'manifest': str(Path(args.manifest).resolve()),
              'manifest_sha256': sha256(Path(args.manifest).resolve()),
              'request': {'product': args.product, 'role': args.role, 'claim': args.claim, 'state': args.state},
              'sources': receipts, 'candidates': ranked, 'errors': errors,
              'status': 'pass' if not errors else 'fail',
              'limitations': ['Selection is deterministic metadata matching, not a quality verdict.',
                              'Proof scope must be reviewed against the exact product capture.',
                              'Rights status is a receipt state, not legal advice.']}
    rendered = json.dumps(result, indent=2) + '\n'
    if args.out:
        Path(args.out).write_text(rendered)
    print(rendered, end='')
    raise SystemExit(1 if errors else 0)


if __name__ == '__main__':
    main()
