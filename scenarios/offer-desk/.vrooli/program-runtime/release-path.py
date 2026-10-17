"""Join the release ladder, enabling urgency and (optionally) one stream's prerequisites."""
import json

inputs = program.inputs()

envelope = {'program': 'offer-desk.release-path', 'version': '1', 'status': 'failed',
            'phase': 'validate', 'inputs': {}, 'signals': {}, 'errors': [], 'evidence': []}

ROW_CAP = 40
SHIPPED = ('TRIGGER_MET', 'RETIRED')


fail = program.fail


guarded = program.guarded


state = {'ladder': [], 'enabling': [], 'prereq': None, 'stream': ''}


def step_validate():
    # phase: validate — inputs only; no binding may be called here.
    stream = inputs.get('stream_node_id') or ''
    if not isinstance(stream, str):
        return fail('failed', 'invalid_input', 'stream_node_id must be a string', 'validate')
    state['stream'] = stream.strip()
    envelope['inputs'] = {'stream_node_id': state['stream'],
                          'include_retired': bool(inputs.get('include_retired', False))}
    return 'collect'


def step_collect():
    # phase: collect — the two board reads run concurrently; prerequisites need a stream id.
    envelope['phase'] = 'collect'
    include_retired = bool(inputs.get('include_retired', False))
    results = gather(
        guarded(lambda: offer_desk.offers.release_ladder(include_retired=include_retired, rows='entries')),
        guarded(lambda: offer_desk.offers.enabling_list(include_retired=include_retired, rows='enabling')),
    )
    reachable = False
    for name, result in zip(('ladder', 'enabling'), results):
        if isinstance(result, Exception):
            status, klass = program.classify(result)
            envelope['errors'].append({'class': klass, 'detail': str(result)[:200], 'where': 'collect:' + name})
            envelope['status'] = status if status == 'unavailable' else 'partial'
        else:
            reachable = True
            state[name] = result.head(ROW_CAP)
            envelope['evidence'].append('offer-desk/offers/release-' + ('ladder' if name == 'ladder' else 'enabling'))
    if not reachable:
        # Every read failed; nothing is known, so the caller must not read this as an empty ladder.
        return fail('unavailable', 'scenario_unreachable', 'no board read succeeded', 'collect')

    if state['stream']:
        try:
            handle = offer_desk.offers.release_prerequisites(stream_node_id=state['stream'], rows='deliverables')
            state['prereq'] = handle.head(ROW_CAP)
            envelope['evidence'].append('offer-desk/offers/release-prerequisites')
        except Exception as exc:
            status, klass = program.classify(exc)
            envelope['errors'].append({'class': klass, 'detail': str(exc)[:200], 'where': 'collect:prerequisites'})
            if envelope['status'] != 'unavailable':
                envelope['status'] = 'partial'
    return 'classify'


def step_classify():
    # phase: classify — a deterministic join; ranking is the board's, never recomputed here.
    envelope['phase'] = 'classify'
    urgency = {}
    for row in state['enabling']:
        node = row.get('node') or {}
        node_id = node.get('id')
        if node_id:
            # protojson omits an int at zero, so an absent derivedUrgency is 0, not unknown.
            urgency[node_id] = int(row.get('derivedUrgency', 0) or 0)

    path = []
    for row in state['ladder']:
        deliverable = row.get('deliverable') or {}
        node_id = deliverable.get('id')
        status = str(deliverable.get('status') or '')
        enablers = row.get('enablers') or []
        blocking = []
        for enabler in enablers:
            enabler_node = enabler if isinstance(enabler, dict) else {}
            inner = enabler_node.get('node') or enabler_node
            enabler_status = str(inner.get('status') or '')
            if enabler_status and enabler_status not in SHIPPED:
                blocking.append({'id': inner.get('id'), 'name': inner.get('name'),
                                 'status': enabler_status,
                                 'urgency': urgency.get(inner.get('id'))})
        path.append({
            'node_id': node_id,
            'name': deliverable.get('name'),
            'status': status,
            'release_rank': int(deliverable.get('releaseRank', 0) or 0),
            'deliverable_class': deliverable.get('deliverableClass'),
            'finish_bar': deliverable.get('finishBar'),
            'actuals_mapped': bool(deliverable.get('actualAccountId')),
            'unlocked_ramps': len(row.get('unlockedRamps') or []),
            'blocked_by': blocking,
            'blocked': bool(blocking),
            'enabling_urgency': urgency.get(node_id),
        })

    ready = [p for p in path if not p['blocked'] and p['status'] not in SHIPPED]
    envelope['signals'] = {
        'ladder_entries': len(path),
        'path': path,
        'next_ready': ready[:5],
        'ready_count': len(ready),
        'blocked_count': sum(1 for p in path if p['blocked']),
        'unmapped_actuals_count': sum(1 for p in path if not p['actuals_mapped']),
        'enabling_ranked': sorted(
            ({'node_id': k, 'urgency': v} for k, v in urgency.items()),
            key=lambda r: -r['urgency'])[:10],
        'prerequisites_requested': bool(state['stream']),
        'prerequisites': state['prereq'],
    }
    if envelope['status'] not in ('partial', 'unavailable'):
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
