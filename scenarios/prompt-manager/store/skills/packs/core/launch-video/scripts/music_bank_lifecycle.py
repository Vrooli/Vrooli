#!/usr/bin/env python3
"""Verify launch-music render binding and prepare exact-hash Library actions.

This tool never calls Music Library. It emits auditable owner-CLI commands only
after the managed render is verified against the frozen production snapshot.
"""
import argparse
import hashlib
import json
from pathlib import Path
import sys

import workflow


def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()


def load(path):
    return json.loads(Path(path).read_text())


def prepare(snapshot, bundle, reserve_plan_path, reservation_path, playback_path, cleanup_path,
            project, revision, consumer, output):
    snapshot, bundle = Path(snapshot).resolve(), Path(bundle).resolve()
    errors = workflow.verify_render_bundle(snapshot, bundle)
    frozen = load(snapshot / 'render-inputs.json')
    music = frozen.get('music_bank')
    if not music or not music.get('track_sha256'):
        errors.append('frozen snapshot has no music-bank exact hash')
    reservation = load(reservation_path)
    plan = load(reserve_plan_path)
    reserved_track = reservation.get('track') or {}
    reservation_id = reservation.get('reservation_id') or reservation.get('reservationId')
    if not reservation_id or reserved_track.get('sha256') != music.get('track_sha256'):
        errors.append('owner CLI reservation receipt does not bind the frozen exact hash')
    if (plan.get('status') != 'owner_cli_admission_required' or
            plan.get('track_sha256') != music.get('track_sha256') or
            plan.get('snapshot_sha256') != sha(snapshot / 'render-inputs.json') or
            plan.get('project') != project or str(plan.get('revision')) != str(revision) or
            plan.get('consumer') != consumer):
        errors.append('reservation plan does not bind this exact hash/project/revision/consumer')
    if errors:
        raise ValueError('; '.join(errors))
    video_sha = sha(bundle / 'render.mp4')
    playback = load(playback_path)
    cleanup = load(cleanup_path)
    if playback.get('video_sha256') != video_sha or playback.get('decoder_status') != 'pass' or playback.get('playback_status') != 'pass':
        errors.append('decoder/playback evidence is missing, failing, or bound to another video hash')
    if cleanup.get('status') != 'clean' or cleanup.get('new_owner_items'):
        errors.append('owner resource cleanup receipt is missing or has outstanding items')
    if errors:
        raise ValueError('; '.join(errors))
    record = {
        'schema_version': 1,
        'status': 'render_verified_owner_cli_pending',
        'project': project,
        'revision': revision,
        'consumer': consumer,
        'reservation_id': reservation_id,
        'reserve_plan_sha256': sha(reserve_plan_path),
        'reservation_receipt_sha256': sha(reservation_path),
        'track_sha256': music['track_sha256'],
        'snapshot_path': str(snapshot),
        'snapshot_receipt_sha256': sha(snapshot / 'render-inputs.json'),
        'render_bundle_path': str(bundle),
        'render_receipt_sha256': sha(bundle / 'receipt.json'),
        'render_sha256': video_sha,
        'playback_receipt_sha256': sha(playback_path),
        'cleanup_receipt_sha256': sha(cleanup_path),
        'owner_cli_actions': {
            'commit_draft': [
                'music-library', 'catalog', 'commit', '--reservation', reservation_id,
                '--consumer', consumer, '--status', 'draft', '--json'
            ],
            'rollback_abandoned_draft': [
                'music-library', 'catalog', 'rollback', '--reservation', reservation_id,
                '--consumer', consumer, '--json'
            ],
            'release_abandoned_reservation': [
                'music-library', 'catalog', 'release', '--reservation', reservation_id,
                '--consumer', consumer, '--json'
            ],
        },
        'acceptance_rule': 'Only the operator may change draft to accepted; never infer acceptance from render or lint.',
        'note': 'No Library mutation was performed by this tool.'
    }
    Path(output).write_text(json.dumps(record, indent=2) + '\n')
    return record


def reserve_plan(snapshot, catalog, track_sha, project, revision, consumer, output, mode='new'):
    """Emit an exact-hash reservation command from a frozen, linted snapshot."""
    snapshot = Path(snapshot).resolve()
    errors = workflow.verify_snapshot(snapshot)
    frozen = load(snapshot / 'render-inputs.json')
    music = frozen.get('music_bank') or {}
    if music.get('track_sha256') != track_sha:
        errors.append('requested hash differs from frozen music selection')
    lint = load(snapshot / music.get('lint_file', '')) if music.get('lint_file') else {}
    if lint.get('status') != 'pass' or lint.get('track_sha256') != track_sha:
        errors.append('frozen music lint does not pass for requested exact hash')
    rows = load(catalog).get('tracks', [])
    row = next((r for r in rows if r.get('sha256') == track_sha), None)
    if not row:
        errors.append('exact hash is absent from supplied catalog snapshot')
    else:
        purposes = row.get('purposes')
        if not isinstance(purposes, list) or 'launch-video' not in purposes:
            errors.append('catalog snapshot does not prove launch-video purpose for this exact hash')
        bank_state = row.get('bank_state')
        if bank_state not in ('available', 'draft', 'used'):
            errors.append('catalog snapshot has no current reservable bank state or track is already reserved')
        if row.get('approval') != 'liked':
            errors.append('exact hash is not operator-liked in supplied catalog snapshot')
        uses = row.get('uses', [])
        same_revision = [u for u in uses if u.get('project') == project and str(u.get('revision')) == str(revision)]
        if same_revision:
            errors.append('exact hash already has a use for this project/revision')
        if mode == 'new' and uses:
            errors.append('new-film selection requires an unused exact hash')
        if mode == 'new' and bank_state != 'available':
            errors.append('new-film selection requires Music Library bank_state available')
        if mode == 're-edit' and not uses:
            errors.append('re-edit mode requires an existing exact-hash use to preserve')
        if mode == 're-edit' and bank_state not in ('draft', 'used'):
            errors.append('re-edit selection requires Music Library bank_state draft or used')
    if errors:
        raise ValueError('; '.join(errors))
    result = {
        'schema_version': 1, 'status': 'owner_cli_admission_required', 'mode': mode,
        'track_sha256': track_sha, 'project': project, 'revision': revision,
        'consumer': consumer, 'catalog_sha256': sha(catalog),
        'snapshot_sha256': sha(snapshot / 'render-inputs.json'),
        'command': ['music-library', 'catalog', 'reserve', '--purpose', 'launch-video',
                    '--project', project, '--revision', revision, '--consumer', consumer,
                    '--sha256', track_sha, '--json'],
        'gate': 'Execute only after current Music Library owner CLI admission and fresh catalog availability check.',
        'note': 'Plan only; no reservation or Library mutation was performed.'
    }
    Path(output).write_text(json.dumps(result, indent=2) + '\n')
    return result


def verify_state(before_path, after_path, track_sha, project, revision, output):
    """Prove one exact hash gained one draft association and nothing else changed."""
    before, after = load(before_path), load(after_path)
    def tracks(doc):
        rows = doc.get('tracks', doc.get('Tracks', []))
        return {r.get('sha256', r.get('sha256'.title())): r for r in rows}
    b, a = tracks(before), tracks(after)
    errors = []
    if set(b) != set(a):
        errors.append('catalog track set changed')
    for digest in sorted(set(b) & set(a)):
        br, ar = b[digest], a[digest]
        if digest != track_sha and br != ar:
            errors.append(f'unselected catalog row changed: {digest}')
    row = a.get(track_sha)
    if not row:
        errors.append('selected exact hash missing after commit')
    else:
        uses = row.get('uses', row.get('Uses', []))
        matches = [u for u in uses if u.get('project') == project and str(u.get('revision')) == str(revision)]
        if len(matches) != 1 or matches[0].get('status') != 'draft':
            errors.append('selected hash does not have exactly one matching draft use')
        if row.get('bank_state') != 'draft':
            errors.append('selected hash Music Library bank_state is not draft after commit')
        before_uses = b[track_sha].get('uses', b[track_sha].get('Uses', []))
        expected_uses = before_uses + matches if len(matches) == 1 else None
        if uses != expected_uses:
            errors.append('selected hash use history changed by more than the one appended draft')
        before_other, after_other = dict(b[track_sha]), dict(row)
        before_other.pop('uses', None); before_other.pop('Uses', None)
        after_other.pop('uses', None); after_other.pop('Uses', None)
        before_other.pop('bank_state', None); before_other.pop('bankState', None)
        after_other.pop('bank_state', None); after_other.pop('bankState', None)
        if before_other != after_other:
            errors.append('selected catalog metadata changed during use transition')
    result = {'status': 'pass' if not errors else 'refused', 'track_sha256': track_sha,
              'project': project, 'revision': revision, 'before_sha256': sha(before_path),
              'after_sha256': sha(after_path), 'errors': errors}
    Path(output).write_text(json.dumps(result, indent=2) + '\n')
    return result


def verify_cleanup_state(before_path, after_path, track_sha, project, revision, action, output):
    """Verify owner-CLI release/rollback without losing unrelated use history."""
    before, after = load(before_path), load(after_path)
    b = {r.get('sha256'): r for r in before.get('tracks', [])}
    a = {r.get('sha256'): r for r in after.get('tracks', [])}
    errors = []
    if set(b) != set(a):
        errors.append('catalog track set changed')
    if track_sha not in b or track_sha not in a:
        errors.append('selected exact hash missing from before/after catalog')
    for digest in sorted(set(b) & set(a)):
        br, ar = b[digest], a[digest]
        if digest != track_sha and br != ar:
            errors.append(f'unselected catalog row changed: {digest}')
        if digest != track_sha:
            continue
        bu, au = br.get('uses', []), ar.get('uses', [])
        target = [u for u in bu if u.get('project') == project and str(u.get('revision')) == str(revision)]
        if action == 'release':
            if bu != au:
                errors.append('release changed recorded use history')
        elif len(target) != 1 or target[0].get('status') != 'draft':
            errors.append('rollback input does not contain exactly one target draft use')
        elif au != [u for u in bu if u is not target[0]]:
            errors.append('rollback changed unrelated use history or failed to remove target draft')
        before_other, after_other = dict(br), dict(ar)
        for key in ('uses', 'Uses', 'bank_state', 'bankState'):
            before_other.pop(key, None); after_other.pop(key, None)
        if before_other != after_other:
            errors.append('selected catalog metadata changed during cleanup')
        remaining = au
        expected_state = ('used' if any(u.get('status') == 'accepted' for u in remaining)
                          else 'draft' if remaining else 'available')
        if ar.get('bank_state') != expected_state:
            errors.append(f'post-cleanup bank_state should be {expected_state}')
    result = {'status': 'pass' if not errors else 'refused', 'action': action,
              'track_sha256': track_sha, 'project': project, 'revision': revision,
              'before_sha256': sha(before_path), 'after_sha256': sha(after_path), 'errors': errors}
    Path(output).write_text(json.dumps(result, indent=2) + '\n')
    return result


def cleanup_receipt(footprint_path, output):
    """Convert the governed footprint diff into a strict clean/unclean receipt."""
    diff = load(footprint_path)
    clean = diff.get('exit_code') == 0 and diff.get('new_count') == 0 and not diff.get('not_checked')
    result = {'schema_version': 1, 'status': 'clean' if clean else 'unclean',
              'footprint_diff_sha256': sha(footprint_path),
              'new_owner_items': diff.get('new_count'),
              'not_checked': diff.get('not_checked', []),
              'evidence': 'research-coordination/scripts/footprint.py diff --baseline B --keep-file K --disk --json',
              'errors': [] if clean else ['footprint diff has new items, unchecked sections, or nonzero exit code']}
    Path(output).write_text(json.dumps(result, indent=2) + '\n')
    return result


def main():
    p = argparse.ArgumentParser(description=__doc__)
    sub = p.add_subparsers(dest='command', required=True)
    q = sub.add_parser('prepare', help='verify render, playback, cleanup and reservation then emit owner-CLI use plan')
    for name in ('snapshot', 'bundle', 'reserve-plan', 'reservation-receipt', 'playback-receipt', 'cleanup-receipt', 'project', 'revision', 'consumer', 'out'):
        q.add_argument('--' + name, required=True)
    q = sub.add_parser('reserve-plan', help='emit exact-hash owner-CLI reservation plan')
    for name in ('snapshot', 'catalog', 'track-sha256', 'project', 'revision', 'consumer', 'out'):
        q.add_argument('--' + name, required=True)
    q.add_argument('--mode', choices=('new', 're-edit'), default='new',
                   help='new requires no uses; re-edit reuses this exact SHA while preserving prior revisions')
    q = sub.add_parser('verify-state', help='prove exact-hash draft transition from CLI exports')
    for name in ('before', 'after', 'track-sha256', 'project', 'revision', 'out'):
        q.add_argument('--' + name, required=True)
    for action in ('release', 'rollback'):
        q = sub.add_parser(f'verify-{action}', help=f'prove owner-CLI {action} changed only selected state')
        for name in ('before', 'after', 'track-sha256', 'project', 'revision', 'out'):
            q.add_argument('--' + name, required=True)
    q = sub.add_parser('cleanup-receipt', help='validate governed footprint diff and write clean receipt')
    q.add_argument('--footprint-diff', required=True)
    q.add_argument('--out', required=True)
    args = p.parse_args()
    try:
        if args.command == 'prepare':
            result = prepare(args.snapshot, args.bundle, args.reserve_plan, args.reservation_receipt,
                             args.playback_receipt, args.cleanup_receipt, args.project,
                             args.revision, args.consumer, args.out)
        elif args.command == 'reserve-plan':
            result = reserve_plan(args.snapshot, args.catalog, args.track_sha256, args.project,
                                  args.revision, args.consumer, args.out, args.mode)
        elif args.command == 'verify-state':
            result = verify_state(args.before, args.after, args.track_sha256, args.project, args.revision, args.out)
        elif args.command in ('verify-release', 'verify-rollback'):
            result = verify_cleanup_state(args.before, args.after, args.track_sha256, args.project,
                                          args.revision, args.command.removeprefix('verify-'), args.out)
        else:
            result = cleanup_receipt(args.footprint_diff, args.out)
        print(json.dumps(result, indent=2))
        return 0 if result.get('status', '').startswith('render_verified') or result.get('status') == 'pass' or result.get('status') == 'clean' else 1
    except (OSError, ValueError, KeyError, json.JSONDecodeError) as exc:
        print(f'refused: {exc}', file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
