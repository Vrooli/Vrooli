#!/usr/bin/env python3
"""File-only launch-video checks. No lifecycle, network, installs, or publishing."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import subprocess
import sys

SKILL = Path(__file__).resolve().parents[1]
STORY = ('promise', 'viewer', 'problem', 'transformation', 'proof', 'takeaway')
CATEGORIES = {'story', 'product clarity', 'composition', 'typography', 'motion',
              'transition', 'audio', 'branding', 'technical', 'claim risk'}


def digest(path):
    with Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def run(args):
    result = subprocess.run(args, capture_output=True, text=True, timeout=240)
    if result.returncode:
        raise ValueError(f'{args[0]} failed ({result.returncode}): {result.stderr[-2000:]}')
    return result


def probe(path):
    return json.loads(run(['ffprobe', '-v', 'error', '-show_streams', '-show_format',
                           '-of', 'json', str(path)]).stdout)


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
    need(plan.get('version') == 1, 'version must be 1')
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
        if s.get('class') in ('evidence', 'presentation'):
            if s.get('capture_mode') == 'ui-fixture':
                need(s.get('proof_scope') == 'ui', f'{label}: fixture proof scope must be ui')
                need(text(s.get('disclosure')), f'{label}: visible demo-data disclosure required')
                file(s.get('capture_report'), label+' fixture capture report')
                file(s.get('schema_validation'), label+' fixture schema validation')
                for claim in s.get('claims', []):
                    need(by_id.get(claim, {}).get('scope') == 'ui',
                         f'{label}: fixture capture cannot prove backend claim {claim}')
            p = file(s.get('media'), label+' media')
            for key in ('capture_instance', 'capture_date', 'source_sha256', 'privacy_review'):
                need(text(s.get(key)), f'{label}: {key} required')
            file(s.get('privacy_review'), label+' privacy review')
            need(s.get('product_motion') is True, f'{label}: real product motion required')
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
        captured = sum(s['duration'] for s in scenes if s.get('id') in evidence and number(s.get('duration')))
        need(number(duration, .1) and captured >= duration * .5,
             'product captures must occupy at least half the film')
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
        if audio.get('variety_requested'):
            styles = audio.get('candidate_styles', [])
            need(isinstance(styles, list) and len({s for s in styles if text(s)}) >= 2,
                 'requested music variety needs distinct styles')
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
            frames = plan.get('styleframes', [])
            need({f.get('role') for f in frames} >= {'opening','hero','detail','ending'}, 'four styleframe roles required')
            for f in frames:
                asset = file(f.get('file'), 'styleframe')
                review(f.get('review'), asset, 'styleframe review')
            rough = file(plan.get('animatic'), 'animatic')
            review(plan.get('animatic_review'), rough, 'animatic review')
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
                report_path = root / plan.get('technical_report', '')
                if report_path.is_file():
                    report = json.loads(report_path.read_text())
                    need(report.get('sha256') == digest(asset) and report.get('decode_ok') is True, 'technical inspection is stale or decode failed')
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


def inspect(video, output, times):
    meta = probe(video)
    duration = float(meta['format']['duration'])
    if not math.isfinite(duration) or not 0 < duration <= 600:
        raise ValueError('inspection supports finite videos up to 600 seconds')
    requested = [float(t) for t in times.split(',')] if times else [min(duration-.04, duration*i/12) for i in range(13)]
    if not 1 <= len(requested) <= 64 or any(not math.isfinite(t) or t < 0 or t >= duration for t in requested):
        raise ValueError('provide 1–64 finite sample times within the video')
    output.mkdir(parents=True, exist_ok=False)
    result = subprocess.run(['ffmpeg', '-v', 'error', '-threads', '2', '-i', str(video),
                             '-f', 'null', '-'], capture_output=True, text=True, timeout=240)
    (output/'decode.log').write_text(result.stderr)
    report = {'video': str(video.resolve()), 'sha256': digest(video), 'metadata': meta,
              'decode_ok': result.returncode == 0 and not result.stderr.strip(),
              'samples': [], 'creative_verdict': 'unreviewed'}
    filters = ['-vf', 'blackdetect=d=0.08:pix_th=0.05']
    if any(s['codec_type'] == 'audio' for s in meta['streams']):
        filters += ['-af', 'silencedetect=n=-50dB:d=0.3,loudnorm=I=-16:TP=-1:LRA=11:print_format=json']
    analysis = run(['ffmpeg', '-hide_banner', '-threads', '2', '-i', str(video), *filters, '-f', 'null', '-'])
    (output/'signal-analysis.log').write_text(analysis.stderr)
    for i,t in enumerate(requested):
        name=f'frame-{i:02d}-{t:06.2f}.jpg'
        run(['ffmpeg','-v','error','-threads','2','-ss',str(t),'-i',str(video),'-frames:v','1','-q:v','2',str(output/name)])
        report['samples'].append({'time':t,'file':name})
    (output/'inspection.json').write_text(json.dumps(report,indent=2)+'\n')
    return report


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    sub=parser.add_subparsers(dest='command',required=True)
    v=sub.add_parser('validate',help='Check a production plan; cannot certify aesthetic quality')
    v.add_argument('plan',type=Path);v.add_argument('--stage',choices=['plan','polish','deliver'],default='plan')
    s=sub.add_parser('shots',help='Retrieve at most four shot recipes by intent')
    s.add_argument('intent');s.add_argument('--limit',type=int,default=4)
    i=sub.add_parser('inspect',help='Decode, probe, and sample an exported file into a new directory')
    i.add_argument('video',type=Path);i.add_argument('--out',type=Path,required=True);i.add_argument('--at',default='')
    args=parser.parse_args()
    try:
        if args.command=='validate':
            plan=json.loads(args.plan.read_text())
            errors=validate(plan,args.plan.resolve().parent,args.stage)
            print(json.dumps({'stage':args.stage,'status':'fail' if errors else 'pass','errors':errors,
                              'limit':'Structural/media checks only; creative and factual review remain required.'},indent=2))
            return bool(errors)
        if args.command=='shots':
            if not 1 <= args.limit <= 12: raise ValueError('limit must be 1–12')
            catalog=json.loads((SKILL/'references/shots.json').read_text())
            query=args.intent.lower().split()
            ranked=sorted(((sum(w in json.dumps(r).lower() for w in query),r) for r in catalog['shots']),key=lambda x:-x[0])
            print(json.dumps([r for score,r in ranked if score][:args.limit],indent=2))
        else:
            r=inspect(args.video,args.out,args.at)
            print(json.dumps({'sha256':r['sha256'],'decode_ok':r['decode_ok'],'samples':len(r['samples']),
                              'report':str(args.out/'inspection.json'),'creative_verdict':'unreviewed'},indent=2))
            return not r['decode_ok']
    except (OSError,ValueError,KeyError,TypeError,AttributeError,subprocess.TimeoutExpired) as e:
        print(json.dumps({'status':'fail','error':str(e)}));return 1
    return 0


if __name__=='__main__':
    sys.exit(main())
