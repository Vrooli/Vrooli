#!/usr/bin/env python3
"""OCR-based visual text-entry and on-screen reading-time gate.

Use this for explicitly authored title/caption cues with a bounded screen
region. It complements, and does not replace, the production-plan hold check.
"""
import argparse
import csv
import io
import hashlib
import json
import shutil
import subprocess
import sys
from pathlib import Path


def digest(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def words(text):
    import re
    return re.findall(r'[a-z0-9]+', str(text).casefold())


def classify_samples(expected_text, samples, *, require_atomic_entrance=False,
                     max_reveal_delay_seconds=1.0, minimum_hold_seconds=1.2,
                     minimum_word_confidence=50.0):
    """Classify partial entrances and measured complete-phrase holds."""
    if (not isinstance(minimum_word_confidence, (int, float))
            or not 50.0 <= minimum_word_confidence <= 100.0):
        raise ValueError('minimum OCR word confidence must be between 50 and 100')
    expected = set(words(expected_text))
    if not expected:
        raise ValueError('expected text cue must contain words')
    normalized = []
    for sample in samples:
        accepted_words = set()
        if 'ocr_tokens' in sample:
            for token in sample.get('ocr_tokens', []):
                confidence = token.get('confidence')
                if (isinstance(confidence, (int, float)) and confidence >= minimum_word_confidence
                        and token.get('text')):
                    accepted_words.update(words(token['text']))
            raw_words = [token.get('text', '') for token in sample.get('ocr_tokens', [])]
        else:
            # Deterministic corpus fixtures provide explicit recognized words.
            raw_words = sample.get('words', [])
            accepted_words.update(words(' '.join(raw_words)))
        matches = expected.intersection(accepted_words)
        complete = expected.issubset(matches)
        partial = bool(matches) and not complete
        normalized.append({**sample, 'words': raw_words,
                           'accepted_words': sorted(accepted_words),
                           'complete': complete, 'partial': partial})
    complete_rows = [row for row in normalized if row['complete']]
    partial_rows = [row for row in normalized if row['partial']]
    findings = []
    if require_atomic_entrance and partial_rows:
        findings.append({'gate': 'partial-text-entrance', 'status': 'fail',
                         'first_partial_seconds': partial_rows[0]['time_seconds']})
    if complete_rows:
        first_complete = complete_rows[0]['time_seconds']
        last_complete = first_complete
        max_hold = 0.0
        current_start = None
        previous = None
        for row in normalized:
            if row['complete']:
                if previous is None or row['time_seconds'] - previous > row['sample_step_seconds'] * 1.5:
                    if current_start is not None:
                        max_hold = max(max_hold, previous - current_start + row['sample_step_seconds'])
                    current_start = row['time_seconds']
                previous = row['time_seconds']
                last_complete = row['time_seconds']
        if current_start is not None:
            step = normalized[-1]['sample_step_seconds']
            max_hold = max(max_hold, previous - current_start + step)
        reveal_delay = first_complete - normalized[0]['time_seconds']
        if reveal_delay > max_reveal_delay_seconds:
            findings.append({'gate': 'text-reveal-time', 'status': 'fail',
                             'first_complete_seconds': first_complete,
                             'reveal_delay_seconds': reveal_delay})
        if max_hold < minimum_hold_seconds:
            findings.append({'gate': 'visible-reading-time', 'status': 'fail',
                             'longest_complete_hold_seconds': round(max_hold, 4)})
    else:
        first_complete, last_complete, reveal_delay, max_hold = None, None, None, 0.0
        findings.append({'gate': 'visible-reading-time', 'status': 'fail',
                         'reason': 'complete phrase was never recognized'})
    return {
        'expected_text': expected_text,
        'first_partial_seconds': partial_rows[0]['time_seconds'] if partial_rows else None,
        'first_complete_seconds': first_complete,
        'last_complete_seconds': last_complete,
        'reveal_delay_seconds': reveal_delay,
        'longest_complete_hold_seconds': round(max_hold, 4),
        'observations': normalized,
        'findings': findings,
        'status': 'fail' if findings else 'pass',
    }


def recognize(crop, tesseract_path):
    import cv2
    ok, png = cv2.imencode('.png', crop)
    if not ok:
        raise ValueError('could not encode OCR crop')
    result = subprocess.run([tesseract_path, 'stdin', 'stdout', '--psm', '6', 'tsv'],
                            input=png.tobytes(), capture_output=True, timeout=10)
    if result.returncode:
        raise ValueError(result.stderr.decode(errors='replace')[-1000:])
    rows = csv.DictReader(io.StringIO(result.stdout.decode(errors='replace')), delimiter='\t')
    return [{'text': row['text'], 'confidence': float(row['conf'])}
            for row in rows if row.get('level') == '5' and row.get('text', '').strip()]


def scan(video, cues, sample_every_frames=3):
    import cv2
    tesseract_path = shutil.which('tesseract')
    if not tesseract_path:
        raise ValueError('tesseract is not installed; no tool installation was attempted')
    cap = cv2.VideoCapture(str(video))
    if not cap.isOpened():
        raise ValueError(f'cannot open video: {video}')
    fps = cap.get(cv2.CAP_PROP_FPS)
    if fps <= 0 or not sample_every_frames:
        cap.release()
        raise ValueError('invalid video fps or sampling interval')
    report_cues = []
    try:
        for cue in cues:
            start, end = cue.get('start_seconds'), cue.get('end_seconds')
            roi = cue.get('region')
            if not isinstance(start, (int, float)) or not isinstance(end, (int, float)) or end <= start:
                raise ValueError(f"{cue.get('id', 'cue')}: invalid cue interval")
            if not isinstance(roi, list) or len(roi) != 4 or any(not isinstance(x, int) for x in roi):
                raise ValueError(f"{cue.get('id', 'cue')}: region must be [x,y,width,height]")
            x, y, width, height = roi
            cap.set(cv2.CAP_PROP_POS_MSEC, start * 1000)
            samples = []
            frame_index = round(start * fps)
            last_frame = round(end * fps)
            while frame_index <= last_frame:
                cap.set(cv2.CAP_PROP_POS_FRAMES, frame_index)
                ok, frame = cap.read()
                if not ok:
                    break
                if x < 0 or y < 0 or width < 1 or height < 1 or x + width > frame.shape[1] or y + height > frame.shape[0]:
                    raise ValueError(f"{cue.get('id', 'cue')}: region outside decoded frame")
                crop = frame[y:y + height, x:x + width]
                tokens = recognize(crop, tesseract_path)
                samples.append({'time_seconds': round(frame_index / fps, 6),
                                'sample_step_seconds': sample_every_frames / fps,
                                'ocr_tokens': tokens})
                frame_index += sample_every_frames
            outcome = classify_samples(
                cue.get('text', ''), samples,
                require_atomic_entrance=cue.get('require_atomic_entrance') is True,
                max_reveal_delay_seconds=cue.get('max_reveal_delay_seconds', 1.0),
                minimum_hold_seconds=cue.get('minimum_hold_seconds', 1.2),
                minimum_word_confidence=cue.get('minimum_word_confidence', 50.0))
            report_cues.append({'id': cue.get('id'),
                               'minimum_word_confidence': cue.get('minimum_word_confidence', 50.0),
                               **outcome})
    finally:
        cap.release()
    return {
        'version': 1, 'video': str(Path(video).resolve()), 'sha256': digest(video),
        'fps': fps, 'sample_every_frames': sample_every_frames,
        'ocr_tool': tesseract_path, 'ocr_page_segmentation_mode': 6,
        'default_minimum_word_confidence': 50.0, 'cues': report_cues,
        'status': 'fail' if any(c['status'] == 'fail' for c in report_cues) else 'pass',
        'limits': ['OCR is a visual cue, not a viewer-comprehension test.',
                   'Only exact normalized tokens at or above the confidence threshold count toward a complete phrase.',
                   'A correct region and expected copy must be supplied from the frozen edit plan.'],
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('video', type=Path)
    parser.add_argument('--cues', type=Path, required=True, help='JSON list of expected text/region cues')
    parser.add_argument('--sample-every-frames', type=int, default=3)
    parser.add_argument('--out', type=Path, required=True, help='new receipt path')
    args = parser.parse_args()
    try:
        if args.out.exists():
            raise ValueError('output receipt already exists')
        cues = json.loads(args.cues.read_text())
        result = scan(args.video, cues, args.sample_every_frames)
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(result, indent=2) + '\n')
        print(json.dumps({'status': result['status'], 'cues': len(result['cues']), 'receipt': str(args.out)}))
        return 0 if result['status'] == 'pass' else 2
    except (OSError, ValueError, TypeError, json.JSONDecodeError, subprocess.SubprocessError) as error:
        print(json.dumps({'status': 'error', 'error': str(error)}), file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
