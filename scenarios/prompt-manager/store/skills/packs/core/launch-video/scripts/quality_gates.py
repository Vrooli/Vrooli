"""Deterministic, file-only launch-video signal gates.

The producer supplies measurements from the exact encoded export. These gates
are deliberately diagnostic: they catch measurable defects without claiming
that a technically clean mix or edit is aesthetically good.
"""
import math
import argparse
import hashlib
import json
import re
import subprocess
from datetime import datetime, timezone
from pathlib import Path
import sys


def _finite(value):
    return isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value)


def evaluate(plan, signals):
    """Return stable findings for reading time, pace, loudness and audio seams.

    `signals` is bound to the exact export hash by the caller and contains
    measured `lufs`, `true_peak_dbtp`, `loop_similarity`, `seam_jump_db`, and
    per-channel 100ms RMS windows in `channel_rms`.
    """
    findings = []
    if not isinstance(plan, dict):
        return [{'gate': 'plan-coverage', 'status': 'fail'}]
    if not isinstance(signals, dict):
        return [{'gate': 'signal-coverage', 'status': 'fail'}]
    scenes = plan.get('scenes', [])
    if not isinstance(scenes, list) or not scenes:
        findings.append({'gate': 'plan-coverage', 'status': 'fail',
                         'reason': 'at least one export scene is required'})
        scenes = []
    for scene in scenes:
        for line in scene.get('text', []):
            words = len(line.get('value', '').split())
            minimum = .8 if words <= 3 else max(1.2, words * .3)
            hold = line.get('hold')
            if not _finite(hold) or hold < minimum or hold > scene.get('duration', 0):
                findings.append({'gate': 'reading-time', 'scene': scene.get('id'), 'status': 'fail'})

    # Rapid cuts are measured over each rolling 10-second window. The threshold
    # is a diagnostic ceiling (12/min); brief action montages remain reviewable.
    starts = sorted(s.get('start') for s in scenes if _finite(s.get('start')))
    for index, start in enumerate(starts):
        count = sum(start <= cut < start + 10 for cut in starts[index + 1:])
        if count > 2:
            findings.append({'gate': 'cut-rate', 'at': start, 'status': 'fail'})
            break

    for key, predicate in (
        ('lufs', lambda x: abs(x - plan.get('audio', {}).get('target_lufs', -16)) <= 2),
        ('true_peak_dbtp', lambda x: x <= -1),
        ('loop_similarity', lambda x: x <= .98),
        ('seam_jump_db', lambda x: x <= 12),
    ):
        value = signals.get(key)
        if not _finite(value) or not predicate(value):
            findings.append({'gate': key.replace('_', '-'), 'status': 'fail'})

    channels = signals.get('channel_rms')
    if not isinstance(channels, list) or not channels:
        findings.append({'gate': 'channel-cut', 'status': 'fail'})
    else:
        for index, windows in enumerate(channels):
            if not isinstance(windows, list) or not windows or any(not _finite(v) or v < .001 for v in windows):
                findings.append({'gate': 'channel-cut', 'channel': index, 'status': 'fail'})

    # Channel variants are checked against the editorial contract maintained
    # in LV1/VARIANT-ENGINEERING.md. Review observations are supplied with the
    # plan; this gate never infers them from silence or a render hash.
    variants = plan.get('channel_cuts')
    if variants is not None:
        required = {'landing', 'X', 'Reddit', 'vertical'}
        by_channel = {row.get('channel'): row for row in variants if isinstance(row, dict)} if isinstance(variants, list) else {}
        if set(by_channel) != required:
            findings.append({'gate': 'channel-cut', 'status': 'fail',
                             'reason': 'each channel needs exactly one cut'})
        hashes = []
        for channel, row in by_channel.items():
            errors = []
            width, height = row.get('width'), row.get('height')
            if not _finite(width) or not _finite(height) or width < 1 or height < 1:
                errors.append('dimensions')
            elif channel in ('landing', 'X') and abs(width / height - 16 / 9) > .02:
                errors.append('landscape-ratio')
            elif channel == 'vertical' and abs(width / height - 9 / 16) > .02:
                errors.append('vertical-ratio')
            if not row.get('burned_captions'):
                errors.append('captions')
            if channel == 'landing' and not row.get('sound_on_reviewed'):
                errors.append('sound-on-review')
            if channel == 'X':
                if not _finite(row.get('hook_seconds')) or row['hook_seconds'] > 2:
                    errors.append('opening-hook')
                if not _finite(row.get('product_proof_seconds')) or row['product_proof_seconds'] > 5:
                    errors.append('early-product-proof')
            if channel == 'Reddit':
                if not row.get('recomposed'):
                    errors.append('feed-recomposition')
                if row.get('qr') is not False:
                    errors.append('qr-disallowed')
                if row.get('unsupported_route_claim') is not False:
                    errors.append('route-claim')
            if channel == 'vertical':
                if not row.get('recomposed') or row.get('center_crop_only') is not False:
                    errors.append('vertical-recomposition')
                if row.get('safe_area_conflicts') != 0:
                    errors.append('safe-area')
                if not row.get('phone_legibility_pass'):
                    errors.append('phone-legibility')
                if row.get('implies_native_mobile') is not False:
                    errors.append('native-mobile-implication')
            digest = row.get('export_sha256')
            if not isinstance(digest, str) or len(digest) != 64 or any(c not in '0123456789abcdef' for c in digest.lower()):
                errors.append('export-hash')
            else:
                hashes.append(digest.lower())
            if errors:
                findings.append({'gate': 'channel-cut', 'channel': channel,
                                 'failed_requirements': errors, 'status': 'fail'})
        if len(hashes) != len(required) or len(set(hashes)) != len(hashes):
            findings.append({'gate': 'channel-cut', 'status': 'fail',
                             'reason': 'each channel must identify its own unique export hash'})
    return findings


def file_sha256(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(block)
    return digest.hexdigest()


def evaluate_export(video, plan, signals):
    """Bind measurements to the exact export and return a durable gate receipt."""
    sha = file_sha256(video)
    findings = evaluate(plan, signals)
    if signals.get('source_sha256') != sha:
        findings.append({'gate': 'source-binding', 'status': 'fail',
                         'reason': 'measurement receipt hash differs from export'})
    return {
        'version': 1, 'checked_at': datetime.now(timezone.utc).isoformat(),
        'video': str(Path(video).resolve()), 'sha256': sha,
        'signals_sha256': signals.get('source_sha256'),
        'status': 'pass' if not findings else 'fail',
        'findings': findings, 'signals': signals,
        'method': 'file-only launch-video quality gates; exact-export hash binding',
        'limits': ['Plan and signal measurements must come from the source owner.',
                   'A technical pass is not an aesthetic or operator approval.'],
    }


def measure_audio_pcm(pcm, sample_rate, *, join_times=(), loop_window_seconds=8,
                     loop_hop_seconds=2, minimum_loop_gap_seconds=16,
                     rms_window_seconds=.1):
    """Measure channel RMS, repeated waveform windows and explicit audio joins."""
    import numpy as np
    values = np.asarray(pcm, dtype=np.float32)
    if values.ndim == 1:
        values = values[:, None]
    if values.ndim != 2 or values.shape[0] == 0 or values.shape[1] == 0:
        raise ValueError('decoded audio must be a non-empty samples-by-channels array')
    if not isinstance(sample_rate, int) or sample_rate < 1000 or not np.isfinite(values).all():
        raise ValueError('invalid sample rate or non-finite audio samples')

    rms_width = max(1, round(sample_rate * rms_window_seconds))
    windows = [values[start:start + rms_width] for start in range(0, len(values), rms_width)]
    channel_rms = [[float(np.sqrt(np.mean(window[:, channel] ** 2)))
                    for window in windows if len(window)]
                   for channel in range(values.shape[1])]

    # Downsample with a box filter to 1kHz for tractable normalized window
    # comparison. A high score indicates near-duplicate audio, not a taste call.
    factor = max(1, round(sample_rate / 1000))
    mono = values.mean(axis=1)
    usable = len(mono) - len(mono) % factor
    down = mono[:usable].reshape(-1, factor).mean(axis=1)
    loop_width = max(1, round(loop_window_seconds * 1000))
    loop_hop = max(1, round(loop_hop_seconds * 1000))
    loop_vectors, loop_starts = [], []
    for start in range(0, len(down) - loop_width + 1, loop_hop):
        segment = down[start:start + loop_width].astype(np.float64)
        segment -= segment.mean()
        norm = np.linalg.norm(segment)
        if norm > 1e-9:
            loop_vectors.append(segment / norm)
            loop_starts.append(start / 1000)
    best = {'similarity': 0.0, 'start_seconds': None, 'match_seconds': None}
    for i in range(len(loop_vectors)):
        for j in range(i + 1, len(loop_vectors)):
            if loop_starts[j] - loop_starts[i] < minimum_loop_gap_seconds:
                continue
            score = float(np.dot(loop_vectors[i], loop_vectors[j]))
            if score > best['similarity']:
                best = {'similarity': score, 'start_seconds': loop_starts[i],
                        'match_seconds': loop_starts[j]}

    joins = []
    join_values = []
    for time in join_times:
        if not _finite(time) or time <= 0 or time >= len(values) / sample_rate:
            raise ValueError('audio join time must be within the decoded stream')
        index = round(time * sample_rate)
        before, after = mono[index - 1], mono[index]
        radius = max(2, sample_rate // 2)
        local = mono[max(0, index - radius):min(len(mono), index + radius)]
        differences = np.abs(np.diff(local))
        relative = index - max(0, index - radius)
        if 0 <= relative - 1 < len(differences):
            differences = np.delete(differences, relative - 1)
        typical = float(np.median(differences)) if len(differences) else 0.0
        jump = float(abs(after - before))
        db = 20 * math.log10(max(jump, 1e-12) / max(typical, 1e-12))
        joins.append({'time_seconds': float(time), 'jump_db_over_local_median': db})
        join_values.append(db)

    return {
        'loop_similarity': best['similarity'] if loop_vectors else None,
        'loop_match': best,
        'seam_jump_db': max(join_values) if join_values else None,
        'seam_measurements': joins,
        'channel_rms': channel_rms,
        'rms_window_seconds': rms_window_seconds,
    }


def measure_video_audio(video, *, join_times=()):
    """Decode exact export audio and measure loop/seam/channel signals + LUFS."""
    video = Path(video)
    probe = subprocess.run(['ffprobe', '-v', 'error', '-select_streams', 'a:0',
                            '-show_entries', 'stream=channels,sample_rate', '-of', 'json', str(video)],
                           capture_output=True, text=True, timeout=30)
    if probe.returncode:
        raise ValueError(probe.stderr[-1000:])
    streams = json.loads(probe.stdout).get('streams', [])
    if not streams:
        raise ValueError('export has no audio stream')
    channels = int(streams[0]['channels'])
    decoded = subprocess.run(['ffmpeg', '-v', 'error', '-i', str(video), '-map', '0:a:0', '-vn',
                              '-acodec', 'pcm_f32le', '-ar', '12000', '-f', 'f32le', '-'],
                             capture_output=True, timeout=240)
    if decoded.returncode:
        raise ValueError(decoded.stderr.decode(errors='replace')[-1000:])
    import numpy as np
    raw = np.frombuffer(decoded.stdout, dtype='<f4')
    if not channels or len(raw) % channels:
        raise ValueError('decoded PCM does not match the encoded channel count')
    pcm = raw.reshape(-1, channels)
    signals = measure_audio_pcm(pcm, 12000, join_times=join_times)
    loud = subprocess.run(['ffmpeg', '-hide_banner', '-v', 'info', '-i', str(video),
                           '-map', '0:a:0', '-af',
                           'loudnorm=I=-16:TP=-1:LRA=11:print_format=json', '-f', 'null', '-'],
                          capture_output=True, text=True, timeout=240)
    matches = re.findall(r'\{\s*"input_i".*?\}', loud.stderr, flags=re.S)
    if loud.returncode or not matches:
        raise ValueError('FFmpeg loudness measurement did not return input metrics')
    result = json.loads(matches[-1])
    signals.update({
        'lufs': float(result['input_i']),
        'true_peak_dbtp': float(result['input_tp']),
        'source_sha256': file_sha256(video),
        'encoded_channels': channels,
    })
    return signals


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('video', type=Path)
    parser.add_argument('--plan', type=Path, required=True)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument('--signals', type=Path,
                        help='JSON measurements bound to this video SHA-256')
    source.add_argument('--measure-audio', action='store_true',
                        help='decode exact export and measure LUFS, loops, joins and per-channel RMS')
    parser.add_argument('--out', type=Path, required=True,
                        help='new JSON receipt path')
    args = parser.parse_args()
    try:
        if args.out.exists():
            raise ValueError('output receipt already exists')
        plan = json.loads(args.plan.read_text())
        if args.measure_audio:
            joins = plan.get('audio', {}).get('join_times', [])
            signals = measure_video_audio(args.video, join_times=joins)
        else:
            loaded = json.loads(args.signals.read_text())
            signals = loaded.get('signals', loaded) if isinstance(loaded, dict) else loaded
        receipt = evaluate_export(args.video, plan, signals)
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(receipt, indent=2) + '\n')
        print(json.dumps({'status': receipt['status'], 'findings': len(receipt['findings']),
                          'receipt': str(args.out)}))
        return 0 if receipt['status'] == 'pass' else 2
    except (OSError, ValueError, TypeError, json.JSONDecodeError) as error:
        print(json.dumps({'status': 'error', 'error': str(error)}), file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
