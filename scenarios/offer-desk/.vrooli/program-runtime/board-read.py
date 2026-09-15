"""Read the ranked offer board and diff it against a caller-supplied baseline snapshot."""
import json

inputs = program.inputs()

envelope = {'program': 'offer-desk.board-read', 'version': '1', 'status': 'failed',
            'phase': 'validate', 'inputs': {}, 'signals': {}, 'errors': [], 'evidence': []}

# Statuses the board assigns a node. Held here so an unrecognised value is reported as
# drift rather than silently bucketed into a known band.
KNOWN_STATUS = ('IDEA', 'ACTIVE', 'TRIGGER_MET', 'RETIRED')
ROW_CAP = 80


guarded = program.guarded


fail = program.fail


state = {'entries': [], 'meta': {}, 'baseline': {}}


def step_validate():
    # phase: validate — inputs only; no binding may be called here.
    raw = inputs.get('baseline') or {}
    if not isinstance(raw, dict):
        return fail('failed', 'invalid_input', 'baseline must be an object', 'validate')
    prior = raw.get('entries') or {}
    if not isinstance(prior, dict):
        return fail('failed', 'invalid_input', 'baseline.entries must be an object of node_id to status', 'validate')
    state['baseline'] = {str(k): str(v) for k, v in prior.items()}
    envelope['inputs'] = {'baseline_entries': len(state['baseline']),
                          'baseline_captured_at': raw.get('captured_at') or ''}
    return 'collect'


def step_collect():
    # phase: collect — one governed read; the board is the single address for team state.
    envelope['phase'] = 'collect'
    result = gather(guarded(lambda: offer_desk.offers.board_show(rows='entries')))[0]
    if isinstance(result, Exception):
        status, klass = program.classify(result)
        return fail(status, klass, result, 'collect')
    try:
        state['entries'] = result.head(ROW_CAP)
        state['truncated'] = result.count() > ROW_CAP
        state['total'] = result.count()
        state['meta'] = result.meta()
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, 'collect')
    envelope['evidence'].append('offer-desk/offers/board-show')
    return 'classify'


def step_classify():
    # phase: classify — a deterministic table; no judgment is needed to diff two snapshots.
    envelope['phase'] = 'classify'
    entries = state['entries']
    snapshot = {}
    unmapped = []
    unknown_status = []
    for row in entries:
        node_id = str(row.get('nodeId') or row.get('node_id') or '')
        if not node_id:
            continue
        status = str(row.get('status') or '')
        snapshot[node_id] = status
        if status and status not in KNOWN_STATUS:
            unknown_status.append({'node_id': node_id, 'status': status})
        # protojson omits an empty repeated field, so absence here means "no gap recorded".
        for gap in row.get('availability') or []:
            if gap.get('reason'):
                unmapped.append({'node_id': node_id, 'title': row.get('title'),
                                 'source': gap.get('source'), 'reason': gap.get('reason')})

    baseline = state['baseline']
    changes = {'added': [], 'removed': [], 'status_changed': []}
    title_of = {}
    for row in entries:
        node_id = str(row.get('nodeId') or row.get('node_id') or '')
        if node_id:
            title_of[node_id] = row.get('title')
    if baseline:
        for node_id, status in snapshot.items():
            was = baseline.get(node_id)
            if was is None:
                changes['added'].append({'node_id': node_id, 'title': title_of.get(node_id), 'status': status})
            elif was != status:
                changes['status_changed'].append({'node_id': node_id, 'title': title_of.get(node_id),
                                                  'from': was, 'to': status})
        for node_id, status in baseline.items():
            if node_id not in snapshot:
                changes['removed'].append({'node_id': node_id, 'was': status})

    counts = {}
    for status in snapshot.values():
        counts[status] = counts.get(status, 0) + 1

    meta = state['meta']
    # An empty string is protojson's zero for a scalar the response did carry, so a blank
    # gap string means "no gap"; the board states its own unavailability in the text.
    gap_text = str(meta.get('defaultAliveGap') or '')
    evaluation = meta.get('evaluation') or {}

    envelope['signals'] = {
        'total_entries': state.get('total', len(snapshot)),
        'materialized': len(snapshot),
        'truncated': bool(state.get('truncated')),
        'status_counts': counts,
        'unknown_status': unknown_status,
        'snapshot': snapshot,
        'baseline_compared': bool(baseline),
        'changes': changes if baseline else None,
        'changes_status': 'compared' if baseline else 'no_baseline',
        'actuals_gaps': {'count': len(unmapped), 'sample': unmapped[:8]},
        'default_alive_gap': gap_text or None,
        'default_alive_available': bool(gap_text) and not gap_text.lower().startswith('unavailable'),
        'posture_source': meta.get('postureSource'),
        'evaluation_last_result': evaluation.get('lastResult'),
        'evaluation_age_seconds': evaluation.get('ageSeconds'),
    }
    if state.get('truncated'):
        envelope['status'] = 'partial'
        envelope['errors'].append({'class': 'row_cap_reached',
                                   'detail': 'board returned more entries than the materialize cap; diff covers the first %d' % ROW_CAP,
                                   'where': 'classify'})
    else:
        envelope['status'] = 'ok'
    return 'report'


def step_report():
    # phase: report — one envelope on every path.
    envelope['phase'] = 'report'
    print(json.dumps(envelope, allow_nan=False, separators=(',', ':')))
    return None


STATES = {'validate': step_validate, 'collect': step_collect,
          'classify': step_classify, 'report': step_report}
state_name = 'validate'
program.run(STATES, state_name)
