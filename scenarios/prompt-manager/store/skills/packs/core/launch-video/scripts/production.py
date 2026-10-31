#!/usr/bin/env python3
"""File-only launch-video checks. No lifecycle, network, installs, or publishing."""
import argparse
import json
import math
import re
from pathlib import Path
import subprocess
import sys
from urllib.parse import urlparse
from product_brief import digest, validate_product_brief, require_current_brief

SKILL = Path(__file__).resolve().parents[1]
STORY = ('promise', 'viewer', 'problem', 'transformation', 'proof', 'takeaway')
CATEGORIES = {'story', 'product clarity', 'composition', 'typography', 'motion',
              'transition', 'audio', 'branding', 'technical', 'claim risk'}



def run(args):
    result = subprocess.run(args, capture_output=True, text=True, timeout=240)
    if result.returncode:
        raise ValueError(f'{args[0]} failed ({result.returncode}): {result.stderr[-2000:]}')
    return result


def probe(path):
    meta = json.loads(run(['ffprobe', '-v', 'error', '-show_streams', '-show_format',
                           '-of', 'json', str(path)]).stdout)
    missing = []
    for stream in meta.get('streams', []):
        if stream.get('codec_type') in ('video', 'audio'):
            try:
                stream_span(stream)
            except (ValueError, TypeError, KeyError, ZeroDivisionError):
                missing.append(stream)
    if missing:
        duration = float(meta.get('format', {}).get('duration', 0))
        format_start = float(meta.get('format', {}).get('start_time', 0))
        if math.isfinite(duration) and 0 < duration <= 600 and math.isfinite(format_start):
            # Read the beginning and actual EOF, rather than trusting WebM's
            # optional DURATION tags or inventing audio length from the container.
            observations = []
            for interval in ('%+1', f'{max(0, duration - 1)}%'):
                observations.append(json.loads(run(['ffprobe', '-v', 'error', '-read_intervals', interval,
                    '-show_packets', '-show_entries',
                    'packet=stream_index,pts_time,duration_time:packet_side_data=side_data_type,skip_samples,discard_padding',
                    '-of', 'json', str(path)]).stdout).get('packets', []))
            for stream in missing:
                span = observed_packet_span(stream, *observations)
                # A missing/overshot EOF seek must not shrink both tracks to the
                # opening and then declare them mutually complete. This checks
                # timing consistency; it never supplies a missing audio length.
                expected_end = format_start + duration
                if span is not None and stream.get('codec_type') == 'video' and abs(span['end'] - expected_end) > .08:
                    span = None
                if span is not None:
                    stream['observed_packet_span'] = span
    return meta


def number(value, minimum=0):
    return isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value) and value >= minimum



def validate(plan, root, stage='plan'):
    """Return failures, not an aesthetic score. References resolve beside the plan."""
    errors = []
    def need(condition, message):
        if not condition:
            errors.append(message)
    def text(value):
        return isinstance(value, str) and bool(value.strip())
    def file(value, label):
        if not text(value):
            errors.append(f'{label}: file reference required')
            return None
        p = root / value
        if not p.is_file() or not p.stat().st_size:
            errors.append(f'{label}: missing or empty file {value}')
            return None
        return p
    def review(ref, asset, label, require_listening=False):
        p = file(ref, label)
        if p is None or asset is None:
            return
        try:
            r = json.loads(p.read_text())
            need(r.get('sha256') == digest(asset), f'{label}: review does not match asset digest')
            for key in ('reviewer', 'method', 'visual_observations', 'audio_observations'):
                need(text(r.get(key)), f'{label}: {key} required')
            need(r.get('verdict') == 'pass', f'{label}: review must pass')
            if require_listening:
                need(r.get('audio_status') == 'auditioned-pass', f'{label}: exported audio must be auditioned')
            if plan.get('version') in (2, 3) and label == 'final-file review':
                need(r.get('review_kind') in ('self-review', 'independent-review', 'operator-review'),
                     f'{label}: review_kind required')
                for section in ('viewer_comprehension', 'craft'):
                    findings = r.get(section, {})
                    need(findings.get('verdict') == 'pass', f'{label}: {section} must pass')
                    need(text(findings.get('method')), f'{label}: {section} method required')
                    rows = findings.get('scenes', [])
                    covered = {row.get('id') for row in rows if text(row.get('observation')) and row.get('verdict') == 'pass'}
                    need(covered >= {scene.get('id') for scene in plan.get('scenes', [])},
                         f'{label}: {section} requires findings for every scene')
                need(text(r.get('opening_comprehension')), f'{label}: opening comprehension required')
                need(text(r.get('campaign_variety')), f'{label}: campaign variety observation required')
            defects = r.get('defects')
            need(isinstance(defects, list), f'{label}: defects list required')
            for d in defects or []:
                need(number(d.get('timestamp')), f'{label}: defect timestamp invalid')
                need(d.get('severity') in ('blocking', 'major', 'minor'), f'{label}: invalid severity')
                need(d.get('category') in CATEGORIES, f'{label}: invalid category')
                for key in ('observation', 'consequence', 'recommendation', 'disposition'):
                    need(text(d.get(key)), f'{label}: defect {key} required')
                need(not (d.get('severity') in ('blocking', 'major') and d.get('disposition') != 'resolved'),
                     f'{label}: unresolved {d.get("severity")} defect')
        except (ValueError, TypeError, AttributeError) as e:
            errors.append(f'{label}: invalid review: {e}')
    need(plan.get('version') in (1, 2, 3), 'version must be 1, 2 or 3')
    current = plan.get('version') in (2, 3)
    if plan.get('version') == 3:
        errors.extend(validate_product_brief(plan, root, stage))
    need(plan.get('kind') in ('launch', 'study'), 'kind must be launch or study')
    if stage != 'plan' and plan.get('kind') == 'study':
        need(plan.get('study_requested') is True and text(plan.get('request_source')),
             'unrequested study cannot proceed to polish')
    profile = plan.get('profile')
    need(profile in ('teaser', 'flagship'), 'profile must be teaser or flagship')
    need(text(plan.get('product')), 'product required')
    story = plan.get('story', {})
    for key in STORY:
        need(text(story.get(key)), f'story.{key} required')
    if current:
        for key in ('viewing_context', 'category', 'setup', 'next_action'):
            need(text(story.get(key)), f'story.{key} required')
        need(plan.get('review_status') in ('draft', 'ready'), 'review_status must be draft or ready')
    safety = plan.get('safety', {})
    file(safety.get('report'), 'safety report')
    need(safety.get('status') in ('verified', 'capture-blocked'), 'safety status required')
    need(safety.get('protected_mutations') == [], 'protected mutations must be empty')
    fmt = plan.get('format', {})
    for key in ('width', 'height', 'fps'):
        need(number(fmt.get(key), 1), f'format.{key} must be positive')
    duration = plan.get('duration')
    need(number(duration, .1), 'duration must be positive')
    claims = plan.get('claims', [])
    ids = [c.get('id') for c in claims]
    need(len(ids) == len(set(ids)) and all(text(i) for i in ids), 'claim IDs must be nonempty and unique')
    by_id = {c.get('id'): c for c in claims}
    for c in claims:
        for key in ('source', 'read_at', 'approved', 'wording', 'qualifiers', 'availability'):
            need(text(c.get(key)), f'claim {c.get("id")}: {key} required (use "none" for no qualifiers)')
        need(c.get('verified') is True, f'claim {c.get("id")}: current behavior/source reconciliation required')
        if c.get('exact_copy'):
            need(c.get('approved') == c.get('wording'), f'claim {c.get("id")}: locked copy changed')
    scenes = plan.get('scenes', [])
    need(bool(scenes), 'at least one scene required')
    scene_ids = [s.get('id') for s in scenes]
    need(len(scene_ids) == len(set(scene_ids)), 'scene IDs must be unique')
    cursor, evidence = 0., []
    for s in scenes:
        label = f'scene {s.get("id")}'
        for key in ('id', 'purpose', 'adds', 'focal_point', 'motion_reason', 'start_state', 'action', 'end_state'):
            need(text(s.get(key)), f'{label}: {key} required')
        start, length = s.get('start'), s.get('duration')
        valid = number(start) and number(length, .1)
        need(valid, f'{label}: invalid timing')
        if valid:
            need(abs(start-cursor) < .04, f'{label}: gap/overlap in editorial slots')
            cursor = start + length
        need(s.get('class') in ('evidence', 'presentation', 'explanatory'), f'{label}: invalid visual class')
        for claim in s.get('claims', []):
            need(claim in by_id, f'{label}: unknown claim {claim}')
        for line in s.get('text', []):
            words = len(line.get('value', '').split())
            hold = line.get('hold')
            floor = .8 if words <= 3 else max(1.2, words*.3)
            need(number(hold, floor), f'{label}: insufficient settled reading hold')
            need(not valid or (number(hold) and hold <= length), f'{label}: hold exceeds scene')
        if current:
            need(text(s.get('visual_intention')), f'{label}: visual_intention required')
            meaning = s.get('meaning', {})
            for key in ('feature', 'sample_input', 'observed_result', 'viewer_inference'):
                need(text(meaning.get(key)), f'{label}: meaning.{key} required (use none when absent)')
            headline = s.get('headline', '').strip().casefold()
            sample = meaning.get('sample_input', '').strip().casefold()
            need(not headline or sample in ('', 'none') or headline != sample,
                 f'{label}: sample input cannot be the feature headline')
        if s.get('class') in ('evidence', 'presentation'):
            if s.get('capture_mode') == 'ui-fixture':
                need(s.get('proof_scope') == 'ui', f'{label}: fixture proof scope must be ui')
                need(text(s.get('disclosure')), f'{label}: disclosure placement required')
                file(s.get('capture_report'), label+' fixture capture report')
                file(s.get('schema_validation'), label+' fixture schema validation')
                for claim in s.get('claims', []):
                    need(by_id.get(claim, {}).get('scope') == 'ui',
                         f'{label}: fixture capture cannot prove backend claim {claim}')
            p = file(s.get('media'), label+' media')
            for key in ('capture_instance', 'capture_date', 'source_sha256', 'privacy_review'):
                need(text(s.get(key)), f'{label}: {key} required')
            file(s.get('privacy_review'), label+' privacy review')
            if current:
                need(s.get('proof_kind') in ('state', 'interaction'), f'{label}: proof_kind must be state or interaction')
                need(s.get('proof_kind') != 'interaction' or s.get('product_motion') is True,
                     f'{label}: interaction proof requires observed product motion')
            if current:
                for key in ('frame', 'cursor', 'settle_evidence', 'trim'):
                    need(text(s.get(key)), f'{label}: capture {key} required')
            if p:
                need(s.get('source_sha256') == digest(p), f'{label}: source digest mismatch')
            crop, footprint = s.get('crop', []), s.get('max_display', [])
            dims_ok = (len(crop) == 4 and all(number(x) for x in crop) and
                       crop[2] > 0 and crop[3] > 0 and len(footprint) == 2 and all(number(x, 1) for x in footprint))
            need(dims_ok, f'{label}: crop [x,y,w,h] and max_display [w,h] required')
            if dims_ok:
                need(crop[2] >= footprint[0] and crop[3] >= footprint[1], f'{label}: insufficient source pixels')
            if stage != 'plan' and p:
                try:
                    media = probe(p)
                    video = next(v for v in media['streams'] if v['codec_type'] == 'video')
                    need(dims_ok and crop[0]+crop[2] <= video['width'] and crop[1]+crop[3] <= video['height'],
                         f'{label}: crop outside encoded source')
                    still = p.suffix.lower() in ('.png', '.jpg', '.jpeg', '.webp') and video.get('codec_name') in ('png', 'mjpeg', 'webp')
                    if still:
                        need(s.get('proof_kind', 'state') == 'state', f'{label}: still image cannot prove an interaction')
                    else:
                        need(float(media['format']['duration'])+.04 >= length, f'{label}: clip shorter than scene; declare a separate hold')
                except (ValueError, KeyError, StopIteration) as e:
                    errors.append(f'{label}: media probe failed: {e}')
            evidence.append(s['id'])
    need(number(duration) and abs(cursor-duration) < .04, 'scene durations do not match total')
    if safety.get('status') == 'capture-blocked':
        need(not evidence, 'capture-blocked plan cannot introduce unverified product footage')
        need(plan.get('kind') == 'study', 'capture-blocked output must be labeled study')
    if plan.get('kind') != 'study':
        need(story.get('proof_scene') in evidence, 'decisive proof must reference real product media')
        if current:
            need(text(story.get('footage_balance')), 'story.footage_balance rationale required')
    audio = plan.get('audio', {})
    need(audio.get('mode') in ('music', 'voice', 'silent'), 'audio mode required')
    need(text(audio.get('direction')), 'audio direction required')
    file(audio.get('selection'), 'audio selection')
    if audio.get('mode') != 'silent':
        need(audio.get('source_kind') in ('governed-generation', 'licensed-reuse', 'original-score'),
             'audio source_kind required')
        if audio.get('source_kind') == 'original-score':
            need(audio.get('original_score_requested') is True and text(audio.get('request_source')),
                 'original score requires an explicit operator request')
        if current and audio.get('variety_requested'):
            need(text(audio.get('variety_rationale')),
                 'requested music variety needs an evidence-based rationale')
    if current and audio.get('mode') != 'silent':
        source_id = audio.get('source_id', '')
        need(text(source_id), 'audio.source_id must identify the actual selected take/source')
        catalog = json.loads((SKILL/'references/examples.json').read_text())
        excluded = catalog.get('excluded_audio_source_ids', []) + audio.get('excluded_source_ids', [])
        need(not any(value and value in source_id for value in excluded), 'selected audio source was explicitly rejected')
    if profile == 'flagship':
        directions = plan.get('directions', [])
        need(len(directions) >= 3, 'flagship requires three creative directions')
        signatures = {(d.get('structure'), d.get('shot_language'), d.get('audio')) for d in directions}
        need(len(signatures) >= 3, 'directions must differ beyond palette/title')
        for d in directions:
            for key in ('id', 'structure', 'shot_language', 'audio', 'rationale'):
                need(text(d.get(key)), f'direction {d.get("id")}: {key} required')
        need(plan.get('chosen_direction') in [d.get('id') for d in directions], 'chosen direction missing')
        if stage != 'plan':
            if current:
                for direction in directions:
                    rough = file(direction.get('rough'), 'direction timed rough')
                    need(text(direction.get('rough_observations')), 'direction timed comparison observations required')
                    if rough:
                        try:
                            need(float(probe(rough)['format']['duration']) > 0, 'direction rough must have measured timing')
                        except (ValueError, KeyError) as error:
                            errors.append(f'direction rough timing unavailable: {error}')
            frames = plan.get('styleframes', [])
            need({f.get('role') for f in frames} >= {'opening','hero','detail','ending'}, 'four styleframe roles required')
            for f in frames:
                asset = file(f.get('file'), 'styleframe')
                review(f.get('review'), asset, 'styleframe review')
            rough = file(plan.get('animatic'), 'animatic')
            review(plan.get('animatic_review'), rough, 'animatic review')
    if current:
        cta = plan.get('cta', {})
        parsed = urlparse(cta.get('url', ''))
        need(parsed.scheme == 'https' and bool(parsed.hostname) and '.' in parsed.hostname,
             'CTA requires a complete public HTTPS URL')
        for key in ('action', 'expected_content', 'availability'):
            need(text(cta.get(key)), f'cta.{key} required')
        if stage == 'deliver':
            target = file(cta.get('destination_report'), 'CTA destination report')
            if target:
                try:
                    report = json.loads(target.read_text())
                    need(report.get('url') == cta.get('url') and report.get('rendered_product_match') is True
                         and text(report.get('checked_at')) and text(report.get('observation')),
                         'CTA destination must verify rendered product content at the exact URL')
                    file(report.get('evidence'), 'CTA rendered destination evidence')
                except (ValueError, TypeError, AttributeError) as e:
                    errors.append(f'invalid CTA destination report: {e}')
            if cta.get('qr'):
                qr = file(cta.get('qr_report'), 'final-video QR report')
                asset_path = root / plan.get('video', '')
                if qr and asset_path.is_file():
                    report = json.loads(qr.read_text())
                    need(report.get('sha256') == digest(asset_path) and report.get('url') == cta.get('url')
                         and report.get('status') == 'pass' and len(set(report.get('times', []))) >= 2
                         and len(set(report.get('widths', []))) >= 2, 'QR report must cover final video at multiple times and sizes')
                    pairs = {(c.get('time'), c.get('width')) for c in report.get('checks', [])
                             if c.get('passed') is True and c.get('decoded') == cta.get('url')}
                    required = {(t, w) for t in report.get('times', []) for w in report.get('widths', [])}
                    need(report.get('inputs_stable') is True and bool(required) and pairs >= required,
                         'QR report contains missing, failed or mismatched final-frame decodes')
    if stage == 'deliver':
        need(plan.get('kind') == 'launch', 'study is not a completed launch demonstration')
        need(story.get('proof_scene') in evidence, 'missing real product proof')
        asset = file(plan.get('video'), 'delivered MP4')
        file(plan.get('poster'), 'poster')
        file(plan.get('technical_report'), 'technical report')
        if asset:
            review(plan.get('review'), asset, 'final-file review', audio.get('mode') != 'silent')
            try:
                media = probe(asset)
                video = next(v for v in media['streams'] if v['codec_type'] == 'video')
                need(video['width'] == fmt.get('width') and video['height'] == fmt.get('height'), 'export dimensions differ from plan')
                num, den = video['avg_frame_rate'].split('/')
                need(abs(float(num)/float(den)-fmt['fps']) < .01, 'export fps differs from plan')
                need(abs(float(media['format']['duration'])-duration) < .15, 'export duration differs from plan')
                need(audio.get('mode') == 'silent' or any(v['codec_type']=='audio' for v in media['streams']), 'export is missing planned audio')
                if audio.get('mode') != 'silent':
                    need(audio_covers_video(media), 'encoded audio stream does not cover the full film')
                report_path = root / plan.get('technical_report', '')
                if report_path.is_file():
                    report = json.loads(report_path.read_text())
                    need(report.get('sha256') == digest(asset) and report.get('decode_ok') is True, 'technical inspection is stale or decode failed')
                    if current:
                        need(report.get('inputs_stable') is True, 'technical inspection inputs changed during verification')
                        if audio.get('mode') != 'silent':
                            metrics = report.get('audio_metrics', {})
                            lufs, peak = metrics.get('lufs'), metrics.get('true_peak_dbtp')
                            target = audio.get('target_lufs', -16)
                            valid_target = isinstance(target, (float, int)) and not isinstance(target, bool) and math.isfinite(target)
                            need(valid_target, 'invalid loudness target')
                            need(isinstance(lufs, (float, int)) and math.isfinite(lufs) and valid_target and abs(lufs-target) <= 2,
                                 'final audio loudness outside target ±2 LUFS')
                            need(isinstance(peak, (float, int)) and math.isfinite(peak) and peak <= -1,
                                 'final audio true peak exceeds -1 dBTP or is unavailable')
            except (ValueError, KeyError, StopIteration, ZeroDivisionError) as e:
                errors.append(f'export verification failed: {e}')
        if profile == 'flagship':
            revisions = plan.get('revisions', [])
            need(bool(revisions) or text(plan.get('no_revision_reason')), 'flagship needs revision evidence or explicit no-defect exception')
            for r in revisions:
                before = file(r.get('before'), 'revision before')
                after = file(r.get('after'), 'revision after')
                file(r.get('notes'), 'revision notes')
                need(text(r.get('defect')), 'revision must name the defect addressed')
                if before and after:
                    need(digest(before) != digest(after), 'revision files are identical')
            if not revisions and asset:
                review(plan.get('second_review'), asset, 'fresh second review', audio.get('mode') != 'silent')
    return errors


def observed_packet_span(stream, beginning, ending):
    """Observed edge timing, including codec sample trimming; not a gap/silence verdict."""
    groups = []
    for packets in (beginning, ending):
        edges = []
        for packet in packets:
            if packet.get('stream_index') != stream.get('index'):
                continue
            try:
                edges.append(packet_edge(stream, packet))
            except (KeyError, TypeError, ValueError, ZeroDivisionError, AttributeError):
                return None
        if not edges:
            return None
        groups.append(edges)
    return {'start': min(x[0] for x in groups[0]), 'end': max(x[1] for x in groups[1]),
            'method': 'ffprobe-beginning-and-eof-packets',
            'sampled_packets': sum(len(x) for x in groups),
            'beginning_packets': len(groups[0]), 'ending_packets': len(groups[1])}


def packet_edge(stream, packet):
    """Return finite packet timing after any explicit codec sample trimming."""
    start = float(packet['pts_time'])
    length = float(packet['duration_time'])
    end = start + length
    for side in packet.get('side_data_list', []):
        skip = float(side.get('skip_samples', 0))
        discard = float(side.get('discard_padding', 0))
        if skip or discard:
            rate = float(stream['sample_rate'])
            if not math.isfinite(rate) or rate <= 0 or skip < 0 or discard < 0:
                raise ValueError('invalid packet sample trimming')
            start += skip / rate
            end -= discard / rate
    if not all(math.isfinite(v) for v in (start, length, end)) or length <= 0 or end <= start:
        raise ValueError('invalid packet timing')
    return start, end


def stream_span(stream):
    """Use stream timing, never container duration as proof of audio coverage."""
    if 'observed_packet_span' in stream:
        span = stream['observed_packet_span']
        start, end = float(span['start']), float(span['end'])
        if not math.isfinite(start) or not math.isfinite(end) or end <= start:
            raise ValueError('invalid observed packet timing')
        return start, end
    start = float(stream.get('start_time', 0))
    if 'duration' in stream:
        length = float(stream['duration'])
    elif 'duration_ts' in stream and 'time_base' in stream:
        num, den = stream['time_base'].split('/')
        length = float(stream['duration_ts']) * float(num) / float(den)
    else:
        raise ValueError('stream duration unavailable')
    if not math.isfinite(start) or not math.isfinite(length) or length <= 0:
        raise ValueError('invalid stream timing')
    return start, start + length


def audio_covers_video(meta, tolerance=.08):
    try:
        videos = [stream_span(s) for s in meta['streams'] if s['codec_type'] == 'video']
        audio = [stream_span(s) for s in meta['streams'] if s['codec_type'] == 'audio']
        return bool(videos and any(a[0] <= min(v[0] for v in videos) + tolerance and
                                  a[1] >= max(v[1] for v in videos) - tolerance for a in audio))
    except (ValueError, TypeError, KeyError, ZeroDivisionError):
        return False


def inspect(video, output, times, cuts=""):
    original = digest(video)
    meta = probe(video)
    duration = float(meta['format']['duration'])
    if not math.isfinite(duration) or not 0 < duration <= 600:
        raise ValueError('inspection supports finite videos up to 600 seconds')
    requested = [float(t) for t in times.split(',')] if times else [min(duration-.04, duration*i/12) for i in range(13)]
    if not 1 <= len(requested) <= 64 or any(not math.isfinite(t) or t < 0 or t >= duration for t in requested):
        raise ValueError('provide 1–64 finite sample times within the video')
    if cuts:
        fps_stream = next(v for v in meta['streams'] if v['codec_type'] == 'video')
        n, d = fps_stream['avg_frame_rate'].split('/')
        step = float(d) / float(n)
        for cut in [float(t) for t in cuts.split(',')]:
            if not math.isfinite(cut) or not 0 <= cut < duration:
                raise ValueError('cut must be within the video')
            requested.extend(max(0, min(duration - step, cut + offset * step)) for offset in (-1, 0, 1))
        requested = sorted(set(requested))
        if len(requested) > 128:
            raise ValueError('too many cut samples; maximum 128 total')
    output.mkdir(parents=True, exist_ok=False)
    result = subprocess.run(['ffmpeg', '-v', 'error', '-threads', '2', '-i', str(video),
                             '-f', 'null', '-'], capture_output=True, text=True, timeout=240)
    (output/'decode.log').write_text(result.stderr)
    report = {'video': str(video.resolve()), 'sha256': original, 'metadata': meta,
              'decode_ok': result.returncode == 0 and not result.stderr.strip(),
              'samples': [], 'creative_verdict': 'unreviewed'}
    filters = ['-vf', 'blackdetect=d=0.08:pix_th=0.05']
    if any(s['codec_type'] == 'audio' for s in meta['streams']):
        filters += ['-af', 'silencedetect=n=-50dB:d=0.3,loudnorm=I=-16:TP=-1:LRA=11:print_format=json']
    analysis = run(['ffmpeg', '-hide_banner', '-threads', '2', '-i', str(video), *filters, '-f', 'null', '-'])
    (output/'signal-analysis.log').write_text(analysis.stderr)
    matches = re.findall(r'\{\s*"input_i".*?\}', analysis.stderr, flags=re.S)
    if matches:
        measured = json.loads(matches[-1])
        report['audio_metrics'] = {key: float(measured[field]) if math.isfinite(float(measured[field])) else None
                                   for key, field in [('lufs', 'input_i'), ('true_peak_dbtp', 'input_tp')]}
        report['silence_starts'] = [float(t) for t in re.findall(r'silence_start: ([0-9.]+)', analysis.stderr)]
        report['silence_ends'] = [float(t) for t in re.findall(r'silence_end: ([0-9.]+)', analysis.stderr)]
    for i,t in enumerate(requested):
        name=f'frame-{i:02d}-{t:06.2f}.jpg'
        run(['ffmpeg','-v','error','-threads','2','-ss',str(t),'-i',str(video),'-frames:v','1','-q:v','2',str(output/name)])
        report['samples'].append({'time':t,'file':name})
    report['audio_covers_video'] = audio_covers_video(meta)
    report['inputs_stable'] = digest(video) == original
    (output/'inspection.json').write_text(json.dumps(report,indent=2)+'\n')
    return report


def retrieve(rows, intent, limit):
    # Search semantic metadata, not shared paths such as product-video-research.
    keys = ('id', 'intent', 'purpose', 'product', 'viewport', 'date', 'build', 'proof_scope', 'lesson', 'notes', 'media')
    query = set(re.findall(r'[a-z0-9]+', intent.casefold()))
    ranked = []
    for row in rows:
        words = set(re.findall(r'[a-z0-9]+', json.dumps([row.get(k) for k in keys]).casefold()))
        score = len(query & words)
        if score:
            ranked.append((score, row))
    return [dict(row) for _, row in sorted(ranked, key=lambda pair: -pair[0])[:limit]]


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    sub=parser.add_subparsers(dest='command',required=True)
    v=sub.add_parser('validate',help='Check a production plan; cannot certify aesthetic quality')
    v.add_argument('plan',type=Path);v.add_argument('--stage',choices=['plan','polish','deliver'],default='plan')
    v.add_argument('--historical',action='store_true',help='Explicitly inspect a retained version-1/2 production')
    s=sub.add_parser('shots',help='Retrieve at most four shot recipes by intent')
    s.add_argument('intent');s.add_argument('--limit',type=int,default=4)
    i=sub.add_parser('inspect',help='Decode, probe, and sample an exported file into a new directory')
    i.add_argument('video',type=Path);i.add_argument('--out',type=Path,required=True);i.add_argument('--at',default='')
    i.add_argument('--cuts',default='',help='Comma-separated cut times; samples before/on/after each cut')
    e=sub.add_parser('examples',help='Retrieve rated examples and retained failure cases')
    e.add_argument('intent');e.add_argument('--limit',type=int,default=4)
    args=parser.parse_args()
    try:
        if args.command=='validate':
            plan=json.loads(args.plan.read_text())
            errors=validate(plan,args.plan.resolve().parent,args.stage)
            errors += require_current_brief(plan, args.plan.resolve().parent, args.historical)
            print(json.dumps({'stage':args.stage,'status':'fail' if errors else 'pass','errors':errors,
                              'limit':'Structural/media checks only; creative and factual review remain required.'},indent=2))
            return bool(errors)
        if args.command in ('shots', 'examples'):
            if not 1 <= args.limit <= 12: raise ValueError('limit must be 1–12')
            catalog=json.loads((SKILL/'references'/('shots.json' if args.command == 'shots' else 'examples.json')).read_text())
            rows = retrieve(catalog[args.command], args.intent, args.limit)
            if args.command == 'examples':
                for row in rows:
                    path = Path(row.get('video') or row.get('asset') or row.get('evidence', '')).expanduser()
                    row['reference_status'] = 'missing' if not path.is_file() else (
                        'hash-mismatch' if row.get('sha256') and digest(path) != row['sha256'] else 'available')
            print(json.dumps(rows,indent=2))
        else:
            r=inspect(args.video,args.out,args.at,args.cuts)
            print(json.dumps({'sha256':r['sha256'],'decode_ok':r['decode_ok'],'samples':len(r['samples']),
                              'report':str(args.out/'inspection.json'),'creative_verdict':'unreviewed'},indent=2))
            return not (r['decode_ok'] and r['inputs_stable'] and (not any(s['codec_type'] == 'audio' for s in r['metadata']['streams']) or r['audio_covers_video']))
    except (OSError,ValueError,KeyError,TypeError,AttributeError,subprocess.TimeoutExpired) as e:
        print(json.dumps({'status':'fail','error':str(e)}));return 1
    return 0


if __name__=='__main__':
    sys.exit(main())
