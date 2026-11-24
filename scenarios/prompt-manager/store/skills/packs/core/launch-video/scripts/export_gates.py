#!/usr/bin/env python3
"""Run exact-export LV3 checks from a production plan and frozen gate cues.

This is a verification command, not a renderer. It reads the selected export,
produces local receipts, and fails closed when cue plans or measurements are
missing. It never publishes, fetches network content, or mutates the export.
"""
import argparse
import hashlib
import json
import subprocess
import cv2
import sys
from datetime import datetime, timezone
from pathlib import Path

import frame_flicker
import quality_gates
import text_entrance
import verify_qr


def digest(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def _write(path, report):
    path.write_text(json.dumps(report, indent=2) + '\n')


def _error_gate(name, error):
    return {'status': 'fail', 'gate': name, 'error': str(error)}


def run(video, plan_file, cues_file, output):
    video, plan_file, cues_file, output = map(Path, (video, plan_file, cues_file, output))
    if output.exists():
        raise ValueError('output directory already exists')
    source_hash, plan_hash, cues_hash = digest(video), digest(plan_file), digest(cues_file)
    plan, cues = json.loads(plan_file.read_text()), json.loads(cues_file.read_text())
    if not isinstance(plan, dict) or not isinstance(cues, dict):
        raise ValueError('production plan and gate cues must be JSON objects')
    declared_video = plan.get('video')
    if not isinstance(declared_video, str) or not declared_video:
        raise ValueError('production plan must declare the exact video path')
    declared_path = Path(declared_video)
    if declared_path.is_absolute() or '..' in declared_path.parts:
        raise ValueError('production plan video path must be a safe relative path')
    if (plan_file.resolve().parent / declared_path).resolve() != video.resolve():
        raise ValueError('selected export does not match the production plan video')
    output.mkdir(parents=True)
    results = {}

    try:
        fps, detections = frame_flicker.scan_video(video)
        report = {'version': 1, 'sha256': source_hash, 'fps': fps,
                  'detections': detections, 'events': frame_flicker.group_events(detections),
                  'status': 'fail' if detections else 'pass'}
        _write(output / 'transient-frame.json', report)
        results['transient-frame'] = report['status']
    except (OSError, ValueError, TypeError, AttributeError, IndexError, subprocess.SubprocessError, cv2.error) as error:
        results['transient-frame'] = 'fail'
        _write(output / 'transient-frame.json', _error_gate('transient-frame', error))

    try:
        fps, windows = frame_flicker.scan_repeated_video(video)
        report = {'version': 1, 'sha256': source_hash, 'fps': fps,
                  'window_seconds': 1.0, 'minimum_transitions': 4,
                  'windows': windows,
                  'events': frame_flicker.group_repeated_windows(windows, fps),
                  'status': 'fail' if windows else 'pass'}
        _write(output / 'temporal-flicker.json', report)
        results['temporal-flicker'] = report['status']
    except (OSError, ValueError, TypeError, AttributeError, IndexError, subprocess.SubprocessError, cv2.error) as error:
        results['temporal-flicker'] = 'fail'
        _write(output / 'temporal-flicker.json', _error_gate('temporal-flicker', error))

    text_cues = cues.get('text_cues')
    if isinstance(text_cues, list) and text_cues:
        try:
            report = text_entrance.scan(video, text_cues,
                                        cues.get('text_sample_every_frames', 3))
            _write(output / 'text-entrance.json', report)
            results['text-entrance'] = report['status'] if report.get('sha256') == source_hash else 'fail'
        except (OSError, ValueError, TypeError, AttributeError, IndexError, subprocess.SubprocessError, cv2.error) as error:
            results['text-entrance'] = 'fail'
            _write(output / 'text-entrance.json', _error_gate('text-entrance', error))
    else:
        results['text-entrance'] = 'fail'
        _write(output / 'text-entrance.json', _error_gate('text-entrance', 'no frozen text_cues supplied'))

    try:
        join_times = plan.get('audio', {}).get('join_times', [])
        signals = quality_gates.measure_video_audio(video, join_times=join_times)
        report = quality_gates.evaluate_export(video, plan, signals)
        _write(output / 'audio-and-plan-quality.json', report)
        results['audio-and-plan-quality'] = report['status']
    except (OSError, ValueError, TypeError, AttributeError, KeyError, IndexError, subprocess.SubprocessError, cv2.error) as error:
        results['audio-and-plan-quality'] = 'fail'
        _write(output / 'audio-and-plan-quality.json', _error_gate('audio-and-plan-quality', error))

    cta = plan.get('cta') if isinstance(plan.get('cta'), dict) else {}
    qr_spec = cues.get('qr')
    if cta.get('qr') is True:
        if (not isinstance(qr_spec, dict) or not isinstance(qr_spec.get('times'), list)
                or not isinstance(qr_spec.get('widths'), list) or not cta.get('url')
                or qr_spec.get('phone_simulation') is not True):
            results['qr'] = 'fail'
            _write(output / 'qr.json', _error_gate('qr', 'CTA QR requires frozen times, widths, a plan URL and phone_simulation=true'))
        else:
            try:
                report = verify_qr.verify(video, cta['url'], qr_spec['times'], qr_spec['widths'],
                                          output / 'qr',
                                          phone_simulation=qr_spec.get('phone_simulation') is True)
                results['qr'] = report['status'] if report.get('sha256') == source_hash else 'fail'
            except (OSError, ValueError, TypeError, AttributeError, IndexError, subprocess.SubprocessError, cv2.error) as error:
                results['qr'] = 'fail'
                _write(output / 'qr.json', _error_gate('qr', error))
    else:
        results['qr'] = 'not-applicable'

    stable = (digest(video) == source_hash and digest(plan_file) == plan_hash
              and digest(cues_file) == cues_hash)
    if not stable:
        results['source-stability'] = 'fail'
    else:
        results['source-stability'] = 'pass'
    receipt = {
        'version': 1, 'checked_at': datetime.now(timezone.utc).isoformat(),
        'video': str(video.resolve()), 'sha256': source_hash,
        'production_plan': str(plan_file.resolve()), 'plan_sha256': plan_hash,
        'gate_cues': str(cues_file.resolve()), 'cues_sha256': cues_hash,
        'results': results,
        'status': 'pass' if all(value in ('pass', 'not-applicable') for value in results.values()) else 'fail',
        'method': 'file-only composed export gates; exact hashes for export, plan and cue contract',
        'limits': ['A gate pass is not an operator aesthetic judgment.',
                   'Text/QR decisions depend on accurate frozen cue plans.',
                   'Frame detections are review candidates; full export decode may need host admission.'],
    }
    _write(output / 'export-gates.json', receipt)
    return receipt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('video', type=Path)
    parser.add_argument('--plan', required=True, type=Path, help='production JSON with scenes/audio/CTA')
    parser.add_argument('--cues', required=True, type=Path, help='frozen JSON with text_cues and optional QR sampling')
    parser.add_argument('--out', required=True, type=Path, help='new output directory')
    args = parser.parse_args()
    try:
        report = run(args.video, args.plan, args.cues, args.out)
        print(json.dumps({'status': report['status'], 'results': report['results'],
                          'receipt': str(args.out / 'export-gates.json')}))
        return 0 if report['status'] == 'pass' else 2
    except (OSError, ValueError, TypeError, json.JSONDecodeError) as error:
        print(json.dumps({'status': 'error', 'error': str(error)}), file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
