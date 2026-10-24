"""Director Swarm portfolio sensor pulse. Read-only; the contract is the caller boundary.

Consolidates the deterministic gathering the portfolio-manager heartbeat does by hand
(stale backlog selection, aging-goal selection, and the unified board reading) into one
typed envelope. It selects, orders and bounds candidates and flags anomalies; the
keep/refresh/supersede verdict and every proposal stay with the caller.
"""
import json
from datetime import datetime, timezone

raw = program.inputs()
now = datetime.now(timezone.utc)

stale_limit = raw.get('stale_limit', 6)
goal_age_days = raw.get('goal_age_days', 14)
goal_limit = raw.get('goal_limit', 3)

# threshold_days must stay an integer in the envelope even when the input is rejected,
# so the failed envelope still conforms to the typed output schema.
threshold_display = goal_age_days if isinstance(goal_age_days, int) and not isinstance(goal_age_days, bool) else 14

FOCUS_GAP = {
    'status': 'cli_only',
    'class': 'no_governed_binding',
    'detail': 'Command Center focus/open-loop ranked findings are not exposed as a governed program binding; read them via `command-center focus` / `command-center open-loop`.',
}

envelope = {
    'program': 'command-center.portfolio-pulse',
    'version': '1',
    'status': 'failed',
    'phase': 'validate',
    'inputs': raw,
    'signals': {
        'generated_at': now.isoformat(),
        'stale_backlog': {'total_stale': 0, 'returned': 0, 'truncated': False, 'selection': 'stale == true, oldest first', 'items': []},
        'aging_goals': {'threshold_days': threshold_display, 'total_active': 0, 'aging_count': 0, 'returned': 0, 'truncated': False, 'items': []},
        'board': {'status': 'unavailable', 'outcome_count': None, 'untrusted': [], 'reading_at': None, 'truncated': False, 'focus_coverage_gap': FOCUS_GAP},
        'attention': [],
        'quality': {'sources_read': 0, 'sources_unavailable': 0, 'stale_backlog_total': 0, 'aging_goals_total': 0},
    },
    'errors': [],
    'evidence': [],
}
signals = envelope['signals']


def error_class(exc):
    text = str(exc)
    lowered = text.lower()
    if 'lease contention' in lowered or 'database is locked' in lowered:
        return 'lease_contention'
    if 'grant' in text or 'run eligible' in text or 'run_eligible' in text:
        return 'refused'
    if any(v in lowered for v in ('unreachable', 'connection refused', 'no running runtime', 'bridge unavailable', 'invalid_port', 'vrooli_not_found')):
        return 'scenario_unreachable'
    if 'deadline' in lowered:
        return 'deadline_exceeded'
    if 'remote' in lowered or '5xx' in lowered or 'internal server error' in lowered:
        return 'remote_error'
    return 'binding_error'


def clip(value, size=180):
    return str(value if value is not None else '')[:size]


def age_days(value):
    try:
        parsed = datetime.fromisoformat(str(value).replace('Z', '+00:00'))
    except (ValueError, TypeError, AttributeError):
        return None
    if parsed.tzinfo is None:
        return None
    return max(0, int((now - parsed).total_seconds() // 86400))


def record_failure(name, binding, exc):
    klass = error_class(exc)
    envelope['errors'].append({'class': klass, 'where': name, 'detail': clip(exc, 180)})
    return klass


def untrusted_reading(trust):
    if isinstance(trust, bool):
        return trust is False
    if isinstance(trust, str):
        return trust.strip().lower() in (
            'untrusted', 'unavailable', 'ghost', 'saturated', 'shelved',
            'unit_mismatch', 'stale', 'missing', 'degraded', 'false')
    return False


def collect_backlog(result):
    if isinstance(result, Exception):
        return record_failure('stale_backlog', 'swarm-manager/backlog/list', result)
    stale = result.filter(lambda r: r.get('stale') is True)
    total = stale.count()
    rows = stale.sort('updated').head(stale_limit)
    items = []
    for r in rows:
        updated = program.select_field(r, 'updated', 'updatedAt')
        ref = (str(r.get('kind') or '') + '/' + str(r.get('name') or '')).strip('/')
        items.append({'ref': ref or clip(r.get('name')), 'title': clip(r.get('title')),
                      'status': r.get('status'), 'priority': r.get('priority'),
                      'updated': updated, 'age_days': age_days(updated)})
    signals['stale_backlog'] = {'total_stale': total, 'returned': len(items),
                                'truncated': total > stale_limit, 'selection': 'stale == true, oldest first', 'items': items}
    signals['quality']['stale_backlog_total'] = total
    envelope['evidence'].append('swarm-manager/backlog/list:stale_backlog')
    return None


def collect_goals(result):
    if isinstance(result, Exception):
        return record_failure('aging_goals', 'swarm-manager/goals/list', result)
    active = result.filter(lambda r: (r.get('goal', r) or {}).get('status') == 'active')
    total_active = active.count()
    candidates = []
    for r in active.head(80):
        g = r.get('goal', r) or {}
        updated = program.select_field(g, 'updated', 'updatedAt')
        ad = age_days(updated)
        if ad is not None and ad >= goal_age_days:
            scope = {k: v for k, v in (g.get('scope') or {}).items() if isinstance(v, (int, float, bool, str))}
            candidates.append({'name': g.get('name'), 'title': clip(g.get('title')),
                               'updated': updated, 'age_days': ad, 'scope': scope})
    candidates.sort(key=lambda x: x['age_days'], reverse=True)
    items = candidates[:goal_limit]
    signals['aging_goals'] = {'threshold_days': goal_age_days, 'total_active': total_active,
                              'aging_count': len(candidates), 'returned': len(items),
                              'truncated': len(candidates) > goal_limit, 'items': items}
    signals['quality']['aging_goals_total'] = len(candidates)
    envelope['evidence'].append('swarm-manager/goals/list:aging_goals')
    return None


def collect_board(result):
    if isinstance(result, Exception):
        signals['board']['status'] = 'unavailable'
        return record_failure('board', 'command-center/walk/read', result)
    rows = result.head(40)
    meta = result.meta()
    untrusted = []
    for row in rows:
        trust = program.select_field(row, 'trust', 'trusted', 'condition', 'verdict', default=None)
        oid = program.select_field(row, 'id', 'outcome', 'metric', 'key', default=None)
        if untrusted_reading(trust) and oid is not None:
            untrusted.append(str(oid))
    signals['board'] = {'status': 'available',
                        'outcome_count': meta.get('total', len(rows)) if isinstance(meta, dict) else len(rows),
                        'untrusted': untrusted[:20],
                        'reading_at': meta.get('generatedAt') if isinstance(meta, dict) else None,
                        'truncated': bool(meta.get('truncated')) if isinstance(meta, dict) else False,
                        'focus_coverage_gap': FOCUS_GAP}
    envelope['evidence'].append('command-center/walk/read:board')
    return None


def classify():
    unavailable = []
    for name, binding, result, fn in SPECS:
        klass = fn(result)
        if klass is not None:
            unavailable.append(name)
    read = 3 - len(unavailable)
    signals['quality']['sources_read'] = read
    signals['quality']['sources_unavailable'] = len(unavailable)
    attention = []
    if signals['stale_backlog']['total_stale'] > 0:
        attention.append('stale_backlog:%d' % signals['stale_backlog']['total_stale'])
    if signals['aging_goals']['aging_count'] > 0:
        attention.append('aging_goals:%d' % signals['aging_goals']['aging_count'])
    if signals['board']['untrusted']:
        attention.append('board_untrusted:%d' % len(signals['board']['untrusted']))
    if unavailable:
        attention.append('sensor_unavailable:' + ','.join(sorted(unavailable)))
    signals['attention'] = attention
    if read == 0:
        envelope['status'] = 'unavailable'
    elif unavailable:
        envelope['status'] = 'partial'
    else:
        envelope['status'] = 'ok'


def report():
    envelope['phase'] = 'report'
    encoded = json.dumps(envelope, ensure_ascii=True, separators=(',', ':'))
    if len(encoded.encode('utf-8')) > 32000:
        signals['stale_backlog']['items'] = []
        signals['aging_goals']['items'] = []
        signals['board']['untrusted'] = []
        envelope['errors'].append({'class': 'output_bound_exceeded', 'where': 'report',
                                   'detail': 'Item detail dropped to stay within the output bound; counts remain authoritative.'})
        encoded = json.dumps(envelope, ensure_ascii=True, separators=(',', ':'))
    print(encoded)


try:
    for value, label in ((stale_limit, 'stale_limit'), (goal_limit, 'goal_limit')):
        if not isinstance(value, int) or isinstance(value, bool) or not 1 <= value <= 20:
            raise ValueError(label + ' must be an integer from 1 to 20')
    if not isinstance(goal_age_days, int) or isinstance(goal_age_days, bool) or not 1 <= goal_age_days <= 120:
        raise ValueError('goal_age_days must be an integer from 1 to 120')
    envelope['phase'] = 'collect'
    guarded = program.guarded
    backlog_res, goals_res, board_res = gather(
        guarded(lambda: swarm_manager.backlog.list(statuses=['backlog'], archived='ARCHIVED_FILTER_EXCLUDE')),
        guarded(lambda: swarm_manager.goals.list()),
        guarded(lambda: command_center.walk.read(limit=40)),
    )
    SPECS = [
        ('stale_backlog', 'swarm-manager/backlog/list', backlog_res, collect_backlog),
        ('aging_goals', 'swarm-manager/goals/list', goals_res, collect_goals),
        ('board', 'command-center/walk/read', board_res, collect_board),
    ]
    envelope['phase'] = 'classify'
    classify()
except Exception as exc:
    envelope['status'] = 'failed'
    envelope['errors'].append({'class': 'invalid_input' if envelope['phase'] == 'validate' else 'kernel_runtime',
                               'where': envelope['phase'], 'detail': clip(exc, 240)})
report()
