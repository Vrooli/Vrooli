"""Command Center morning walk preparation. Read-only; contract is the caller boundary."""
import json
import hashlib
import re
from datetime import datetime, timezone

try:
    inputs
except NameError:
    inputs = {}

PHASES = [('1', 'Open floor', []), ('2', 'Retrospective', ['outcomes', 'recent_work', 'director_notes']),
          ('3', 'Portfolio decisions', ['portfolio', 'pending_work', 'portfolio_handoff']),
          ('4', 'Strategist decisions', ['strategist_handoff']),
          ('5', 'Monetization decisions', ['monetization_notes']),
          ('5.3', 'Marketing decisions', ['marketing_notes']),
          ('5.5', 'Meta-optimization decisions', ['meta_focus', 'meta_notes']),
          ('5.7', 'Infrastructure decisions', ['infra_focus', 'infra_notes']),
          ('6', 'Outside-Vrooli signals', []), ('7', 'Big picture ideation', ['outcomes', 'portfolio']),
          ('8', 'Actions', []), ('9', 'Wrap-up', [])]
now = datetime.now(timezone.utc)
envelope = {'program': 'command-center.vision-walk-prep', 'version': '3', 'status': 'failed',
            'phase': 'validate', 'inputs': inputs, 'signals': {'generated_at': now.isoformat(),
            'sources': {}, 'phases': [], 'checkpoint': None}, 'errors': [], 'evidence': []}
limit = inputs.get('limit', 5)
max_age = inputs.get('max_age_hours', 36)
channel = inputs.get('channel', 'operator')
kind_suffix = '-test' if channel == 'test' else ''


def error_class(exc):
    text = str(exc)
    if 'no handoff found' in text:
        return 'missing_evidence'
    if 'grant' in text or 'run eligible' in text or 'run_eligible' in text:
        return 'refused'
    if any(v in text.lower() for v in ('unreachable', 'connection refused', 'no running runtime', 'bridge unavailable')):
        return 'scenario_unreachable'
    if 'deadline' in text.lower():
        return 'deadline_exceeded'
    return 'binding_error'


def timestamp(value):
    try:
        parsed = datetime.fromisoformat(value.replace('Z', '+00:00'))
        return parsed if parsed.tzinfo is not None else None
    except (ValueError, TypeError, AttributeError):
        return None


def age_state(value):
    observed = timestamp(value)
    if observed is None:
        return 'unknown'
    seconds = (now - observed).total_seconds()
    if seconds < -300:
        return 'future'
    return 'stale' if seconds > max_age * 3600 else 'fresh'


def clip(value, size=700):
    return str(value or '')[:size]


def guarded(call):
    def run():
        try:
            return call()
        except Exception as exc:
            return exc
    return run


def journal_rows(handle):
    rows = handle.head(limit)
    return [{'id': r.get('id'), 'observed_at': r.get('createdAt'), 'freshness': age_state(r.get('createdAt')),
             'kind': r.get('kind'), 'text': clip(r.get('body')), 'text_truncated': len(r.get('body', '')) > 700}
            for r in rows]


def work_rows(handle):
    return [{'ref': r.get('kind', '') + '/' + r.get('name', ''), 'title': clip(r.get('title'), 180),
             'status': r.get('status'), 'priority': r.get('priority'), 'observed_at': r.get('updated'),
             'freshness': age_state(r.get('updated'))} for r in handle.head(limit)]


def bounded_evidence(value, depth=0):
    if isinstance(value, str):
        return value[:500]
    if isinstance(value, list):
        return [bounded_evidence(v, depth + 1) for v in value[:8]] if depth < 5 else {'omitted': True}
    if isinstance(value, dict):
        return {k: bounded_evidence(v, depth + 1) for k, v in list(value.items())[:30]} if depth < 5 else {'omitted': True}
    return value


def collect():
    specs = [
        ('outcomes', 'command-center/walk/read', lambda: command_center.walk.read(limit=40), 'outcomes'),
        ('portfolio', 'swarm-manager/goals/list', lambda: swarm_manager.goals.list(), 'goals'),
        ('pending_work', 'swarm-manager/backlog/list', lambda: swarm_manager.backlog.list(statuses=['review', 'blocked']), 'work'),
        ('recent_work', 'swarm-manager/backlog/list', lambda: swarm_manager.backlog.list(statuses=['completed']), 'work'),
        ('portfolio_handoff', 'prompt-manager/team/handoff-latest', lambda: prompt_manager.team.handoff_latest(team_id='director-swarm', agent_id='portfolio-manager'), 'handoff'),
        ('strategist_handoff', 'prompt-manager/team/handoff-latest', lambda: prompt_manager.team.handoff_latest(team_id='director-swarm', agent_id='outcome-strategist'), 'handoff'),
        ('meta_focus', 'meta-optimization-manager/focus/next', lambda: meta_optimization_manager.focus.next(), 'focus'),
        ('infra_focus', 'infrastructure-manager/focus/next', lambda: infrastructure_manager.focus.next(rows='findings'), 'focus'),
        ('checkpoint', 'source-ledger/journal/list', lambda: source_ledger.journal.list(scope='team:director-swarm', kind='walk-checkpoint' + kind_suffix, newest_first=True, limit=1), 'checkpoint'),
        ('previous_handoff', 'prompt-manager/team/handoff-latest', lambda: prompt_manager.team.handoff_latest(team_id='director-swarm', agent_id='vision-walk-prep'), 'legacy'),
    ]
    specs.append(('previous_briefing', 'source-ledger/journal/list', lambda: source_ledger.journal.list(scope='team:director-swarm', kind='vision-walk-briefing' + kind_suffix, newest_first=True, limit=1), 'checkpoint'))
    for name, scope in [('director_notes', 'director-swarm'), ('monetization_notes', 'monetization'),
                        ('marketing_notes', 'marketing-crew'), ('meta_notes', 'meta-optimization'), ('infra_notes', 'infra-health')]:
        specs.append((name, 'source-ledger/journal/list', lambda scope=scope: source_ledger.journal.list(scope='team:' + scope, newest_first=True, limit=limit), 'journal'))
    results = gather(*[guarded(s[2]) for s in specs])
    for spec, result in zip(specs, results):
        name, binding, _, kind = spec
        source = {'binding': binding, 'read_at': now.isoformat(), 'status': 'unavailable', 'rows': []}
        envelope['signals']['sources'][name] = source
        if isinstance(result, Exception):
            klass = error_class(result)
            source['reason'] = klass
            envelope['errors'].append({'class': klass, 'where': name, 'detail': clip(result, 180)})
            continue
        try:
            meta = result.meta()
            if kind in ('handoff', 'legacy'):
                data = meta.get('data')
                if not isinstance(data, dict) or not isinstance(data.get('content'), str):
                    raise ValueError('handoff has no content object')
                source.update(status='available', observed_at=data.get('updatedAt') or data.get('timestamp'),
                              text=clip(data['content'], 1400), text_truncated=len(data['content']) > 1400,
                              content_sha256=hashlib.sha256(data['content'].encode('utf-8')).hexdigest())
                source['freshness'] = age_state(source['observed_at'])
                if kind == 'legacy':
                    content = data['content']
                    match = re.search(r'^## Walk Checkpoint[ \t]*\r?$(?:\n|$)', content, re.MULTILINE)
                    section = content[match.start():] if match else None
                    if section:
                        offset = match.end() - match.start()
                        following = re.search(r'^#{1,2} ', section[offset:], re.MULTILINE)
                        if following:
                            section = section[:following.start() + offset]
                    source['legacy_checkpoint'] = section
                    source.pop('text', None)
            elif kind == 'checkpoint':
                rows = result.head(1)
                source.update(status='empty' if not rows else 'available', rows=rows)
            elif kind == 'journal':
                rows = journal_rows(result)
                source.update(status='available' if rows else 'empty', rows=rows, truncated=len(rows) == limit,
                              truncation_reason='latest window; older entries remain in the ledger')
            elif kind == 'work':
                # Filter and order inside the Handle before materialization.
                shaped = result.sort('updated', reverse=(name != 'pending_work'))
                source.update(status='available' if result.count() else 'empty', total=result.count(), rows=work_rows(shaped), truncated=result.count() > limit, selection='oldest pending first; includes older unresolved work' if name == 'pending_work' else 'latest completed first')
            elif kind == 'goals':
                active = result.filter(lambda r: r.get('goal', r).get('status') == 'active')
                source.update(status='available' if active.count() else 'empty', total=active.count(), truncated=active.count() > limit,
                              rows=[{'name': r.get('goal', r).get('name'), 'title': clip(r.get('goal', r).get('title'), 180),
                                     'scope': {k: v for k, v in r.get('scope', {}).items() if isinstance(v, (int, float, bool))}} for r in active.head(limit)])
            elif kind == 'outcomes':
                source.update(status='available', rows=result.head(40), total=meta.get('total', result.count()),
                              truncated=meta.get('truncated', False), observed_at=meta.get('generatedAt'))
            else:
                # Preserve the provider's ranking and qualifications, not a new composite score.
                source.update(status='available' if result.count() else 'empty', rows=result.head(limit),
                              truncated=result.count() > limit, total=result.count(), provider_evidence={k:v for k,v in meta.items() if k not in ('items','findings')})
                raw = json.dumps(source, ensure_ascii=True)
                bounded = bounded_evidence(source)
                bounded['payload_truncated'] = json.dumps(bounded, ensure_ascii=True) != raw
                source.clear()
                source.update(bounded)
            envelope['evidence'].append(binding + ':' + name)
        except Exception as exc:
            source.update(status='unavailable', reason='invalid_evidence', rows=[])
            envelope['errors'].append({'class': 'invalid_evidence', 'where': name, 'detail': clip(exc, 180)})


def checkpoint():
    sources = envelope['signals']['sources']
    current = sources['checkpoint']
    if current['status'] == 'unavailable':
        return {'status': 'unavailable', 'reason': current.get('reason')}
    if current['rows']:
        entry = current['rows'][0]
        ref = entry.get('id')
        body = entry.get('body', '')
        try:
            value = json.loads(body)
            if not isinstance(value, dict) or value.get('state') not in ('active', 'completed', 'abandoned') or not value.get('walk_id'):
                raise ValueError('checkpoint requires walk_id and state')
            if value['state'] == 'active' and (not value.get('resume_phase') or not value.get('content')):
                raise ValueError('active checkpoint requires resume_phase and content')
            return {'status': value['state'], 'entry_id': ref, 'scope': 'team:director-swarm', 'checkpoint': value}
        except (ValueError, TypeError) as exc:
            envelope['errors'].append({'class': 'invalid_checkpoint', 'where': 'checkpoint', 'detail': clip(exc, 180)})
            return {'status': 'invalid', 'entry_id': ref, 'reason': 'invalid_checkpoint'}
    legacy = sources['previous_handoff']
    if legacy.get('legacy_checkpoint'):
        return {'status': 'legacy', 'content': legacy['legacy_checkpoint'], 'source': 'prompt-manager/team/handoff-latest'}
    if legacy['status'] == 'unavailable' and legacy.get('reason') != 'missing_evidence':
        return {'status': 'unavailable', 'reason': 'legacy_handoff_unreadable'}
    return {'status': 'none'}


def compare_previous():
    sources = envelope['signals']['sources']
    previous = sources['previous_briefing']
    delta = {'status':'no_baseline', 'baseline_entry_id':None, 'sources':{},
             'rule':'Compare observed stable identities only. Not observed never means resolved; source and row bounds remain authoritative.'}
    envelope['signals']['changes'] = delta
    if previous['status'] == 'unavailable':
        delta['status']='unavailable'; return
    if not previous['rows']: return
    entry=previous['rows'][0]; delta['baseline_entry_id']=entry.get('id')
    try:
        saved=json.loads(entry.get('body',''))
        old=saved['envelope']['signals']['sources']
        if not isinstance(old,dict): raise ValueError('invalid baseline sources')
    except (ValueError,KeyError,TypeError):
        delta['status']='invalid_baseline';return
    delta['status']='compared'
    def keyed(source):
        return {str(r.get('id') or r.get('ref') or r.get('name') or r.get('gap',{}).get('id')):r for r in source.get('rows',[]) if r.get('id') or r.get('ref') or r.get('name') or r.get('gap',{}).get('id')}
    for name,current in sources.items():
        if name in ('checkpoint','previous_handoff','previous_briefing'):continue
        prior=old.get(name)
        if not isinstance(prior,dict):delta['sources'][name]={'status':'no_source_baseline'};continue
        a,b=keyed(prior),keyed(current)
        def meaningful(row):
            return {k:v for k,v in row.items() if k not in ('observedAt','observed_at')}
        observed_changed=[k for k in b if k in a and meaningful(b[k])!=meaningful(a[k])]
        refreshed=[k for k in b if k in a and b[k]!=a[k] and meaningful(b[k])==meaningful(a[k])]
        delta['sources'][name]={'status':'compared' if current['status']!='unavailable' else 'unavailable',
          'newly_observed':sorted(b.keys()-a.keys())[:40], 'changed':observed_changed[:40], 'refreshed':refreshed[:40],
          'not_observed':sorted(a.keys()-b.keys())[:40],
          'condition_changed':current.get('status')!=prior.get('status'),
          'bounded':bool(current.get('truncated') or prior.get('truncated') or current.get('payload_truncated') or prior.get('payload_truncated'))}
        if name in ('portfolio_handoff', 'strategist_handoff'):
            comparison = {'status': 'unknown', 'reason': 'prior_content_unavailable'}
            if current['status'] == 'unavailable':
                comparison = {'status': 'unavailable', 'reason': current.get('reason')}
            elif prior.get('status') == 'available':
                prior_digest = prior.get('content_sha256')
                # Older snapshots can be compared only when they retain the full text.
                if not prior_digest and prior.get('text_truncated') is False and isinstance(prior.get('text'), str):
                    prior_digest = hashlib.sha256(prior['text'].encode('utf-8')).hexdigest()
                if prior_digest:
                    comparison = {'status': 'unchanged' if prior_digest == current['content_sha256'] else 'changed'}
                else:
                    comparison['reason'] = 'prior_full_content_missing'
            delta['sources'][name]['content_change'] = comparison
            delta['sources'][name]['bounded'] |= bool(current.get('text_truncated') or prior.get('text_truncated'))


def classify():
    envelope['signals']['checkpoint'] = checkpoint()
    compare_previous()
    for name in ('checkpoint', 'previous_handoff', 'previous_briefing'):
        envelope['signals']['sources'].pop(name, None)
    # The owner currently offers this composed sensor only as a CLI operation.
    # Do not reproduce its durable-product attribution algorithm in a program.
    envelope['signals']['fleet_health'] = {'status': 'read_elsewhere', 'command': 'prompt-manager team heartbeat-fleet-health --json',
                                          'reason': 'no_governed_binding'}
    for phase, title, names in PHASES:
        envelope['signals']['phases'].append({'phase': phase, 'title': title, 'sources': names,
            'mode': 'evidence' if names else 'conversation'})
    sources = envelope['signals']['sources'].values()
    available = sum(s['status'] in ('available', 'empty') for s in sources)
    gaps = sum(s['status'] == 'unavailable' for s in sources)
    stale = sum(s.get('freshness') in ('stale', 'unknown', 'future') or any(r.get('freshness') in ('stale', 'unknown', 'future') for r in s.get('rows', [])) for s in sources)
    envelope['signals']['quality'] = {'readable_sources': available, 'unavailable_sources': gaps, 'stale_or_undated_sources': stale,
                                     'manual_supplements': 1, 'phase_count': len(PHASES)}
    cp = envelope['signals']['checkpoint']['status']
    envelope['status'] = 'unavailable' if not available else ('partial' if gaps or stale or cp in ('unavailable', 'invalid') else 'ok')


def report():
    envelope['phase'] = 'report'
    encoded = json.dumps(envelope, ensure_ascii=True, separators=(',', ':'))
    if len(encoded.encode('utf-8')) > 60000:
        # Never print a cut-off JSON object or silently cut a checkpoint.
        envelope['status'] = 'failed'
        sizes = {k: len(json.dumps(v)) for k,v in envelope['signals'].get('sources', {}).items()}
        envelope['signals'] = {'generated_at': now.isoformat(), 'reason': 'output_bound_exceeded', 'source_bytes': sizes}
        envelope['errors'].append({'class': 'output_bound_exceeded', 'where': 'report', 'detail': 'No truncated briefing is usable.', 'source_bytes': {k: len(json.dumps(v)) for k,v in envelope.get('signals', {}).get('sources', {}).items()}})
        encoded = json.dumps(envelope, ensure_ascii=True, separators=(',', ':'))
    print(encoded)


try:
    if channel not in ('operator', 'test'):
        raise ValueError('channel must be operator or test')
    if not isinstance(limit, int) or isinstance(limit, bool) or not 1 <= limit <= 8:
        raise ValueError('limit must be an integer from 1 to 8')
    if not isinstance(max_age, int) or isinstance(max_age, bool) or not 1 <= max_age <= 36:
        raise ValueError('max_age_hours must be an integer from 1 to 36')
    envelope['phase'] = 'collect'
    collect()
    envelope['phase'] = 'classify'
    classify()
except Exception as exc:
    envelope['status'] = 'failed'
    envelope['errors'].append({'class': 'invalid_input' if envelope['phase'] == 'validate' else 'kernel_runtime', 'where': envelope['phase'], 'detail': clip(exc, 240)})
report()
