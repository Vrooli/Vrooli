#!/usr/bin/env python3
"""Bounded file workflow: freeze declared render inputs and join final verification.

No network, model jobs, lifecycle, dependencies, publishing, or arbitrary shell.
Use existing render/capture owners, then verify their actual outputs here.
"""
import argparse
import datetime
import hashlib
import subprocess
import json
from pathlib import Path
import shutil
import sys
from production import digest, inspect, validate
from product_brief import require_current_brief


def freeze(plan_file, output):
    raw_plan = plan_file.read_bytes()
    original = hashlib.sha256(raw_plan).hexdigest()
    plan = json.loads(raw_plan)
    root = plan_file.resolve().parent
    entries = plan.get('render_inputs', [])
    if not 1 <= len(entries) <= 512:
        raise ValueError('declare 1–512 render_inputs files relative to production.json')
    if len(set(entries)) != len(entries):
        raise ValueError('duplicate render inputs')
    records = []
    total = 0
    for name in entries:
        relative = Path(name)
        path = root / relative
        if name == 'render-inputs.json' or relative.is_absolute() or '..' in relative.parts or path.is_symlink() or not path.is_file():
            raise ValueError(f'render input must be a regular relative file: {name}')
        if not path.resolve().is_relative_to(root):
            raise ValueError(f'render input escapes production root: {name}')
        size = path.stat().st_size
        total += size
        if total > 2 * 1024**3:
            raise ValueError('render snapshot exceeds 2 GiB; select a smaller production input set')
        records.append({'file': name, 'sha256': digest(path), 'bytes': size})
    output.mkdir(parents=True, exist_ok=False)
    for row in records:
        destination = output / row['file']
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(root / row['file'], destination)
        if digest(destination) != row['sha256'] or digest(root / row['file']) != row['sha256']:
            raise ValueError(f'input changed during snapshot: {row["file"]}')
    if digest(plan_file) != original:
        raise ValueError('plan changed during snapshot')
    report = {'version': 1, 'created_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
              'plan_sha256': original, 'bytes': total, 'inputs': records,
              'instruction': 'Render from this snapshot. Never mutate snapshot inputs. Verify this receipt after render.',
              'limits': ['Only explicitly declared files are frozen', 'No creative, dependency or lifecycle certification']}
    (output / 'render-inputs.json').write_text(json.dumps(report, indent=2) + '\n')
    return report


def verify_snapshot(snapshot, plan_sha256=None):
    receipt = json.loads((snapshot / 'render-inputs.json').read_text())
    errors = []
    if plan_sha256 is not None and receipt.get('plan_sha256') != plan_sha256:
        errors.append('snapshot belongs to a different production plan')
    for row in receipt['inputs']:
        relative = Path(row['file'])
        path = snapshot / relative
        if relative.is_absolute() or '..' in relative.parts or not path.resolve().is_relative_to(snapshot.resolve()):
            errors.append(f'unsafe snapshot input: {relative}')
        elif not path.is_file() or digest(path) != row['sha256']:
            errors.append(f'changed snapshot input: {relative}')
    return errors


def verify_render_bundle(snapshot, bundle):
    """Join a managed render to this exact local snapshot, not a filename cache."""
    receipt = json.loads((bundle / 'receipt.json').read_text())
    request = receipt.get('request', {})
    errors = verify_snapshot(snapshot)
    if receipt.get('status') != 'completed':
        errors.append('managed render did not complete')
    if Path(request.get('input_dir', '')).resolve() != snapshot.resolve():
        errors.append('managed render used a different snapshot directory')
    actual = {str(p.relative_to(snapshot)): digest(p) for p in snapshot.rglob('*') if p.is_file()}
    declared = request.get('inputs', [])
    if len(declared) != len(actual) or {r.get('path'): r.get('sha256') for r in declared} != actual:
        errors.append('managed render inputs differ from the complete snapshot')
    video = bundle / 'render.mp4'
    if not video.is_file() or digest(video) != receipt.get('output_sha256'):
        errors.append('managed render output hash mismatch')
    return errors


def verify(plan_file, output, snapshot=None, render_bundle=None, historical=False):
    original = digest(plan_file)
    plan = json.loads(plan_file.read_text())
    root = plan_file.resolve().parent
    errors = validate(plan, root, 'polish')
    errors += require_current_brief(plan, root, historical)
    if snapshot:
        errors += verify_snapshot(snapshot, original)
    if render_bundle:
        if snapshot is None:
            errors.append('managed render verification requires --snapshot')
        else:
            errors += verify_render_bundle(snapshot, render_bundle)
    output.mkdir(parents=True, exist_ok=False)
    if errors:
        result = {'status': 'fail', 'stage': 'polish', 'errors': errors, 'plan_sha256': original}
    else:
        cuts = ','.join(str(scene['start']) for scene in plan['scenes'][1:])
        try:
            inspection = inspect(root / plan['video'], output / 'technical', '', cuts)
            # Keep the submitted plan unchanged; retain this resolved delivery view.
            checked = dict(plan, technical_report=str((output / 'technical/inspection.json').resolve()))
            errors = validate(checked, root, 'deliver')
            if snapshot:
                errors += verify_snapshot(snapshot, original)
            if render_bundle and snapshot:
                errors += verify_render_bundle(snapshot, render_bundle)
            if digest(plan_file) != original:
                errors.append('plan changed during verification')
            result = {'status': 'fail' if errors else 'pass', 'stage': 'deliver', 'errors': errors,
                      'plan_sha256': original, 'video_sha256': inspection['sha256'],
                      'technical_report': checked['technical_report'],
                      'limits': ['Review declarations remain subject to critique', 'No publication or new aesthetic acceptance']}
            (output / 'resolved-production.json').write_text(json.dumps(checked, indent=2) + '\n')
        except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as error:
            result = {'status': 'fail', 'stage': 'technical', 'plan_sha256': original, 'errors': [str(error)]}
    (output / 'workflow.json').write_text(json.dumps(result, indent=2) + '\n')
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='command', required=True)
    f = sub.add_parser('freeze');f.add_argument('plan', type=Path);f.add_argument('--out', type=Path, required=True)
    v = sub.add_parser('verify');v.add_argument('plan', type=Path);v.add_argument('--out', type=Path, required=True);v.add_argument('--snapshot', type=Path);v.add_argument('--render-bundle', type=Path)
    v.add_argument('--historical', action='store_true', help='Inspect a retained version-1/2 production')
    args = parser.parse_args()
    try:
        result = freeze(args.plan, args.out) if args.command == 'freeze' else verify(args.plan, args.out, args.snapshot, args.render_bundle, args.historical)
        print(json.dumps({'status': result.get('status', 'pass'), 'receipt': str(args.out), 'errors': result.get('errors', [])}))
        return result.get('status') == 'fail'
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(json.dumps({'status': 'fail', 'error': str(error)}));return 1


if __name__ == '__main__':
    sys.exit(main())
