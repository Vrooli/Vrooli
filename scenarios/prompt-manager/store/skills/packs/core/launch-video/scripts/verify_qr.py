#!/usr/bin/env python3
"""Verify QR content in compressed final video; no network or camera claims."""
import argparse
import datetime
import json
from pathlib import Path
import subprocess
import sys
from urllib.parse import urlparse
from production import digest, probe


def verify(video, url, times, widths, output):
    import cv2  # Existing approved environment only; absence is an explicit error.
    parsed = urlparse(url)
    if parsed.scheme != 'https' or not parsed.hostname or '.' not in parsed.hostname:
        raise ValueError('expected URL must be a complete public HTTPS URL')
    if not 2 <= len(times) <= 8 or not 2 <= len(widths) <= 8:
        raise ValueError('provide 2–8 times and 2–8 playback widths')
    duration = float(probe(video)['format']['duration'])
    if any(not 0 <= t < duration for t in times) or any(not 160 <= w <= 4096 for w in widths):
        raise ValueError('times must be inside the video; widths must be 160–4096')
    original = digest(video)
    output.mkdir(parents=True, exist_ok=False)
    checks = []
    for index, time in enumerate(times):
        frame = output / f'frame-{index:02d}.png'
        subprocess.run(['ffmpeg', '-v', 'error', '-threads', '2', '-ss', str(time), '-i', str(video),
                        '-frames:v', '1', str(frame)], check=True, capture_output=True, timeout=60)
        raster = cv2.imread(str(frame))
        if raster is None:
            raise ValueError(f'no decoded frame at {time}')
        for width in widths:
            scaled = cv2.resize(raster, (width, round(raster.shape[0] * width / raster.shape[1])), interpolation=cv2.INTER_AREA)
            decoded, _, _ = cv2.QRCodeDetector().detectAndDecode(scaled)
            checks.append({'time': time, 'width': width, 'decoded': decoded, 'passed': decoded == url, 'frame': frame.name})
    stable = digest(video) == original
    report = {'version': 1, 'checked_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
              'sha256': original, 'url': url, 'times': times, 'widths': widths, 'checks': checks,
              'inputs_stable': stable, 'status': 'pass' if stable and all(c['passed'] for c in checks) else 'fail',
              'method': 'FFmpeg final-video decode; full-frame area resize; OpenCV QR detector; no crop or sharpening',
              'limits': ['No physical phone-camera test', 'No destination uptime or availability guarantee']}
    (output / 'qr.json').write_text(json.dumps(report, indent=2) + '\n')
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('video', type=Path)
    parser.add_argument('--url', required=True)
    parser.add_argument('--at', required=True, help='Comma-separated stationary hold times')
    parser.add_argument('--widths', default='1920,1280,960,640,390')
    parser.add_argument('--out', type=Path, required=True, help='New receipt directory')
    args = parser.parse_args()
    try:
        result = verify(args.video, args.url, [float(t) for t in args.at.split(',')],
                        [int(w) for w in args.widths.split(',')], args.out)
        print(json.dumps({'status': result['status'], 'checks': len(result['checks']), 'report': str(args.out / 'qr.json')}))
        return result['status'] != 'pass'
    except (OSError, ValueError, ImportError, subprocess.SubprocessError) as error:
        print(json.dumps({'status': 'fail', 'error': str(error)}))
        return 1


if __name__ == '__main__':
    sys.exit(main())
