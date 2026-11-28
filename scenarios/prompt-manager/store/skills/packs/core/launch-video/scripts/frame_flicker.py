#!/usr/bin/env python3
"""Detect one-frame flashes in decoded exports without re-encoding them.

The detector looks for a spatially coherent change that appears in one frame
and returns to the preceding frame's state immediately. Sustained motion and
ordinary cuts are reported separately and are not called flicker.
"""
from __future__ import annotations

import argparse
from collections import deque
import hashlib
import json
import math
import subprocess
import sys
from pathlib import Path

import cv2
import numpy as np


def digest(path: Path) -> str:
    h = hashlib.sha256()
    with path.open('rb') as f:
        for block in iter(lambda: f.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def _gray(frame: np.ndarray) -> np.ndarray:
    return cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY) if frame.ndim == 3 else frame


def detect_frames(frames, fps: float, *, pixel_delta: int = 28,
                  return_tolerance: int = 2, min_area: float = .001,
                  min_component: int = 24, max_static_flow: float = .5):
    """Return transient flashes with frame/time and localized changed area.

    Input is an iterable of decoded BGR or gray frames. This pure function is
    intentionally shared by the media CLI and planted-array tests.
    """
    if not math.isfinite(fps) or fps <= 0:
        raise ValueError('fps must be positive')
    it = iter(frames)
    try:
        prev = _gray(next(it))
        cur = _gray(next(it))
    except StopIteration:
        return []
    hits = []
    index = 1
    for nxt in it:
        nxt = _gray(nxt)
        if cur.shape != prev.shape or cur.shape != nxt.shape:
            raise ValueError('frame dimensions changed during video')
        into = cv2.absdiff(cur, prev) >= pixel_delta
        out = cv2.absdiff(cur, nxt) >= pixel_delta
        stable = cv2.absdiff(prev, nxt) <= return_tolerance
        transient = (into & out & stable).astype(np.uint8)
        # Remove isolated codec noise, preserving actual coherent glyph or
        # panel flashes. A 3px opening cannot erase text-scale defects.
        transient = cv2.morphologyEx(transient, cv2.MORPH_OPEN,
                                     np.ones((3, 3), np.uint8))
        count, labels, stats, _ = cv2.connectedComponentsWithStats(transient, 8)
        flow = cv2.calcOpticalFlowFarneback(prev, nxt, None, .5, 3, 15, 3, 5, 1.2, 0)
        components = []
        for i in range(1, count):
            x, y, w, h, component_area = map(int, stats[i])
            if component_area < min_component:
                continue
            # A feature that moves through its neighborhood is ordinary
            # animation, even when antialiasing makes its edge look transient.
            local_flow = flow[y:y+h, x:x+w]
            median_flow = float(np.median(np.linalg.norm(local_flow, axis=2)))
            if median_flow <= max_static_flow:
                components.append(component_area)
        area = sum(components)
        fraction = area / float(cur.size)
        if fraction >= min_area:
            hits.append({
                'frame': index, 'timestamp_seconds': round(index / fps, 6),
                'transient_pixels': area, 'frame_fraction': round(fraction, 7),
                'components': len(components),
                'peak_component_pixels': max(components, default=0),
            })
        prev, cur = cur, nxt
        index += 1
    return hits



def detect_repeated_flicker(frames, fps: float, *, pixel_delta: int = 28,
                            min_transitions: int = 4, window_seconds: float = 1.0,
                            min_area: float = .001, min_component: int = 24,
                            max_static_flow: float = .5):
    """Find repeated high-contrast changes in the same stationary pixels.

    A single flash creates two transitions and does not pass the default four;
    sustained motion is rejected by the first-to-last optical-flow check. The
    rolling window keeps memory bounded for long exports.
    """
    if not math.isfinite(fps) or fps <= 0 or not math.isfinite(window_seconds) or window_seconds <= 0:
        raise ValueError('fps and window duration must be positive')
    if min_transitions < 4:
        raise ValueError('at least four transitions are required for repeated flicker')
    window_frames = max(4, round(fps * window_seconds))
    ring = deque(maxlen=window_frames + 1)
    transition_maps = deque(maxlen=window_frames)
    transition_counts = None
    expected_shape = None
    hits = []
    for index, frame in enumerate(frames):
        current = _gray(frame)
        if expected_shape is None:
            expected_shape = current.shape
            transition_counts = np.zeros(current.shape, dtype=np.uint16)
        elif current.shape != expected_shape:
            raise ValueError('frame dimensions changed during video')
        if ring:
            if len(transition_maps) == window_frames:
                transition_counts -= transition_maps.popleft()
            active = (cv2.absdiff(current, ring[-1]) >= pixel_delta).astype(np.uint8)
            transition_maps.append(active)
            transition_counts += active
        ring.append(current)
        if len(transition_maps) != window_frames:
            continue
        mask = (transition_counts >= min_transitions).astype(np.uint8)
        if not np.any(mask):
            continue
        mask = cv2.morphologyEx(mask, cv2.MORPH_OPEN, np.ones((3, 3), np.uint8))
        count, labels, stats, _ = cv2.connectedComponentsWithStats(mask, 8)
        candidate_components = [(i, tuple(map(int, stats[i])) )
                                for i in range(1, count) if stats[i, cv2.CC_STAT_AREA] >= min_component]
        if not candidate_components:
            continue
        first, last = ring[0], ring[-1]
        flow = cv2.calcOpticalFlowFarneback(first, last, None, .5, 3, 15, 3, 5, 1.2, 0)
        components = []
        transitions_by_component = []
        for component, (x, y, w, h, area) in candidate_components:
            region = labels[y:y+h, x:x+w] == component
            local_flow = flow[y:y+h, x:x+w]
            median_flow = float(np.median(np.linalg.norm(local_flow[region], axis=1)))
            if median_flow > max_static_flow:
                continue
            components.append(area)
            transitions_by_component.append(int(np.max(transition_counts[y:y+h, x:x+w][region])))
        flicker_pixels = sum(components)
        fraction = flicker_pixels / float(current.size)
        if components and fraction >= min_area:
            hits.append({'first_frame': index - window_frames,
                         'last_frame': index, 'start_seconds': round((index-window_frames)/fps, 6),
                         'end_seconds': round(index/fps, 6),
                         'max_transitions_in_window': max(transitions_by_component),
                         'flicker_pixels': flicker_pixels,
                         'frame_fraction': round(fraction, 7),
                         'components': len(components)})
    return hits



def group_repeated_windows(windows, fps):
    """Merge overlapping rolling windows into concise repeated-flicker intervals."""
    if not windows:
        return []
    ordered = sorted(windows, key=lambda item: item['start_seconds'])
    groups = [[ordered[0]]]
    for row in ordered[1:]:
        if row['start_seconds'] <= groups[-1][-1]['end_seconds'] + 1 / fps:
            groups[-1].append(row)
        else:
            groups.append([row])
    return [{
        'start_seconds': group[0]['start_seconds'],
        'end_seconds': group[-1]['end_seconds'],
        'window_count': len(group),
        'max_transitions_in_window': max(row['max_transitions_in_window'] for row in group),
        'peak_flicker_pixels': max(row['flicker_pixels'] for row in group),
    } for group in groups]


def scan_repeated_video(video: Path, *, analysis_width=480, **settings):
    """Decode an export into the bounded-memory repeated-flicker detector."""
    cap = cv2.VideoCapture(str(video))
    if not cap.isOpened():
        raise ValueError(f'cannot open video: {video}')
    fps = cap.get(cv2.CAP_PROP_FPS)
    def decoded():
        while True:
            ok, frame = cap.read()
            if not ok:
                return
            if frame.shape[1] > analysis_width:
                height = round(frame.shape[0] * analysis_width / frame.shape[1])
                frame = cv2.resize(frame, (analysis_width, height), interpolation=cv2.INTER_AREA)
            yield frame
    try:
        hits = detect_repeated_flicker(decoded(), fps, **settings)
    finally:
        cap.release()
    return fps, hits


def scan_video(video: Path, *, pixel_delta=28, return_tolerance=2,
               min_area=.001, min_component=24, analysis_width=480,
               max_static_flow=.5):
    cap = cv2.VideoCapture(str(video))
    if not cap.isOpened():
        raise ValueError(f'cannot open video: {video}')
    fps = cap.get(cv2.CAP_PROP_FPS)
    if not isinstance(analysis_width, int) or analysis_width < 64:
        cap.release()
        raise ValueError('analysis width must be an integer of at least 64 pixels')
    def decoded():
        while True:
            ok, frame = cap.read()
            if not ok:
                return
            if frame.shape[1] > analysis_width:
                height = round(frame.shape[0] * analysis_width / frame.shape[1])
                frame = cv2.resize(frame, (analysis_width, height), interpolation=cv2.INTER_AREA)
            yield frame
    try:
        hits = detect_frames(decoded(), fps, pixel_delta=pixel_delta,
                             return_tolerance=return_tolerance,
                             min_area=min_area, min_component=min_component,
                             max_static_flow=max_static_flow)
    finally:
        cap.release()
    return fps, hits


def group_events(hits, bridge_frames=1):
    """Group adjacent flagged frames into one timestamped flicker interval."""
    if not hits:
        return []
    groups = [[hits[0]]]
    for hit in hits[1:]:
        if hit['frame'] - groups[-1][-1]['frame'] <= bridge_frames + 1:
            groups[-1].append(hit)
        else:
            groups.append([hit])
    return [{
        'start_seconds': group[0]['timestamp_seconds'],
        'end_seconds': group[-1]['timestamp_seconds'],
        'first_frame': group[0]['frame'], 'last_frame': group[-1]['frame'],
        'flagged_frames': len(group),
        'peak_transient_pixels': max(h['transient_pixels'] for h in group),
        'largest_component_pixels': max(h['peak_component_pixels'] for h in group),
    } for group in groups]


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('video', type=Path)
    p.add_argument('--out', type=Path, required=True, help='new JSON receipt path')
    p.add_argument('--pixel-delta', type=int, default=28)
    p.add_argument('--return-tolerance', type=int, default=7)
    p.add_argument('--min-area', type=float, default=.001)
    p.add_argument('--min-component', type=int, default=24)
    p.add_argument('--max-static-flow', type=float, default=.5,
                   help='ignore candidate regions moving faster between neighbors')
    p.add_argument('--analysis-width', type=int, default=480,
                   help='downscale decoded frames to this width before comparison')
    p.add_argument('--temporal-flicker', action='store_true',
                   help='also decode a rolling 1-second window for repeated stationary flicker')
    a = p.parse_args()
    try:
        before = digest(a.video)
        fps, hits = scan_video(a.video, pixel_delta=a.pixel_delta,
                               return_tolerance=a.return_tolerance,
                               min_area=a.min_area, min_component=a.min_component,
                               analysis_width=a.analysis_width,
                               max_static_flow=a.max_static_flow)
        repeated = []
        if a.temporal_flicker:
            _, repeated = scan_repeated_video(a.video, analysis_width=a.analysis_width)
        stable = digest(a.video) == before
        report = {
            'version': 1, 'input': str(a.video.resolve()), 'sha256': before,
            'inputs_stable': stable, 'fps': fps,
            'analysis_width': a.analysis_width,
            'algorithm': 'localized-one-frame-return',
            'thresholds': {'pixel_delta': a.pixel_delta,
                           'return_tolerance': a.return_tolerance,
                           'min_area_fraction': a.min_area,
                           'min_component_pixels': a.min_component},
            'motion_filter': {'max_static_flow_pixels_per_frame': a.max_static_flow},
            'detections': hits,
            'events': group_events(hits),
            'temporal_flicker_windows': repeated,
            'temporal_flicker_events': group_repeated_windows(repeated, fps) if repeated else [],
            'temporal_flicker_scanned': a.temporal_flicker,
            'temporal_flicker_config': ({'window_seconds': 1.0, 'min_transitions': 4,
                                         'pixel_delta': 28, 'minimum_area_fraction': .001,
                                         'minimum_component_pixels': 24, 'max_static_flow': .5}
                                        if a.temporal_flicker else None),
            'status': 'pass' if stable and not hits and not repeated else 'fail',
            'limits': ['One-frame return scan and repeated stationary-pixel scan are separate signals; camera motion and subtle flicker may require review.',
                       'A detection is a review cue, not an aesthetic verdict.'],
        }
        a.out.parent.mkdir(parents=True, exist_ok=True)
        a.out.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps({'status': report['status'], 'detections': len(hits),
                          'flicker_windows': len(repeated),
                          'flicker_events': len(report['temporal_flicker_events']),
                          'receipt': str(a.out)}))
        return 0 if report['status'] == 'pass' else 2
    except (OSError, ValueError, cv2.error) as e:
        print(json.dumps({'status': 'error', 'error': str(e)}), file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
