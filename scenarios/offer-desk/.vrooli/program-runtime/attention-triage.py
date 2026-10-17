"""Shortlist board entries deterministically, then label the shortlist with one batched call."""
import json

inputs = program.inputs()

envelope = {'program': 'offer-desk.attention-triage', 'version': '1', 'status': 'failed',
            'phase': 'validate', 'inputs': {}, 'signals': {}, 'errors': [], 'evidence': []}

LABELS = ['operator_decision', 'evidence_gap', 'routine', 'stale_idea']
BOARD_CAP = 80
DEFAULT_MAX = 24


fail = program.fail


def as_int(value, fallback):
    try:
        return int(value)
    except (TypeError, ValueError):
        return fallback


def parse_label(row):
    if not isinstance(row, dict) or row.get('validated') is not True:
        return None
    try:
        parsed = json.loads(row.get('valueJson', ''))
    except (TypeError, ValueError):
        return None
    return parsed if parsed in LABELS else None


state = {'entries': [], 'proposals': [], 'shortlist': [], 'max_items': DEFAULT_MAX}


def step_validate():
    # phase: validate — inputs only; no binding may be called here.
    max_items = as_int(inputs.get('max_items', DEFAULT_MAX), DEFAULT_MAX)
    if max_items < 1 or max_items > 40:
        return fail('failed', 'invalid_input', 'max_items must be between 1 and 40', 'validate')
    state['max_items'] = max_items
    envelope['inputs'] = {'max_items': max_items}
    return 'collect'


def step_collect():
    # phase: collect — the board and its open promotion proposals.
    envelope['phase'] = 'collect'
    try:
        handle = offer_desk.offers.board_show(rows='entries')
        state['entries'] = handle.head(BOARD_CAP)
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, 'collect')
    envelope['evidence'].append('offer-desk/offers/board-show')

    try:
        proposals = offer_desk.offers.gates_proposals(rows='proposals')
        state['proposals'] = proposals.head(BOARD_CAP)
        state['proposals_read'] = True
        envelope['evidence'].append('offer-desk/offers/gates-proposals')
    except Exception as exc:
        status, klass = program.classify(exc)
        # protojson omits an empty repeated field entirely, so a response with no open
        # proposals carries no selectable rows at all. "rows must be one of: <none>" is
        # therefore a real reading of zero proposals, not a failed read. Any other
        # ambiguity is genuine drift and still degrades the program.
        if klass == 'ambiguous_response' and 'must be one of: <none>' in str(exc):
            state['proposals'] = []
            state['proposals_read'] = True
            envelope['evidence'].append('offer-desk/offers/gates-proposals')
        else:
            state['proposals_read'] = False
            envelope['errors'].append({'class': klass, 'detail': str(exc)[:200], 'where': 'collect:proposals'})
            envelope['status'] = 'partial'
    return 'classify'


def step_classify():
    # phase: classify — deterministic shortlist first; inference only for the real judgment.
    envelope['phase'] = 'classify'
    proposal_nodes = set()
    for row in state['proposals']:
        node_id = row.get('nodeId') or row.get('node_id')
        if node_id:
            proposal_nodes.add(str(node_id))

    shortlist = []
    for entry in state['entries']:
        node_id = str(entry.get('nodeId') or entry.get('node_id') or '')
        status = str(entry.get('status') or '')
        reason = str(entry.get('rankReason') or '')
        gaps = [g.get('reason') for g in (entry.get('availability') or []) if g.get('reason')]
        # Deterministic inclusion rules; each names why the row is a candidate.
        why = None
        if status == 'TRIGGER_MET':
            why = 'trigger fired and the node is not promoted'
        elif status == 'ACTIVE' and gaps:
            why = 'earning node with unreadable actuals'
        elif node_id in proposal_nodes:
            why = 'open promotion proposal'
        if why:
            shortlist.append({'node_id': node_id, 'title': entry.get('title'), 'status': status,
                              'rank_reason': reason, 'gaps': gaps, 'why_candidate': why})
    shortlist = shortlist[:state['max_items']]
    state['shortlist'] = shortlist

    envelope['signals'] = {
        'board_entries': len(state['entries']),
        'open_proposals': len(state['proposals']) if state.get('proposals_read') else None,
        'proposals_read': bool(state.get('proposals_read')),
        'shortlist_size': len(shortlist),
        'shortlist': shortlist,
        'labelled': [],
        'inference_calls': 0,
    }

    if not shortlist:
        # Nothing deterministic qualified, so no inference is spent. This is a real
        # reading of a quiet board, not an unavailable one.
        envelope['signals']['triage_reason'] = 'no_candidate'
        if envelope['status'] != 'partial':
            envelope['status'] = 'ok'
        return 'report'

    corpus = ['%s [%s] %s | gaps: %s' % (c['title'], c['status'], c['rank_reason'],
                                         ', '.join(c['gaps']) or 'none') for c in shortlist]
    try:
        batch_handle = ai.batch(
            corpus,
            {'type': 'string', 'enum': LABELS},
            'Label each offer board row for the monetization team. '
            'Use operator_decision when only the operator can settle it, evidence_gap when a '
            'missing measurement blocks the decision, stale_idea when the row was captured and '
            'never planned against, and routine when the team can act without the operator.',
            role='classify.fast',
        )
        rows = batch_handle.head(1)
        if len(rows) != 1 or not isinstance(rows[0], dict):
            raise ValueError('batch response must contain one object')
        batch = rows[0]
        results = batch.get('results') if isinstance(batch.get('results'), list) else []
        envelope['signals']['usage'] = batch.get('usage')
        envelope['signals']['inference_calls'] = 1
        labelled = []
        for index, candidate in enumerate(shortlist):
            label = parse_label(results[index]) if index < len(results) else None
            labelled.append({'node_id': candidate['node_id'], 'title': candidate['title'],
                             'why_candidate': candidate['why_candidate'],
                             'label': label, 'abstained': label is None})
        envelope['signals']['labelled'] = labelled
        envelope['signals']['abstained'] = sum(1 for r in labelled if r['abstained'])
        envelope['evidence'].append('ai-gateway/inference/run-batch')
    except Exception as exc:
        status, klass = program.classify(exc)
        # The deterministic shortlist survives a failed classification and is still useful.
        envelope['errors'].append({'class': klass, 'detail': str(exc)[:200], 'where': 'classify:batch'})
        envelope['signals']['triage_reason'] = 'classification_unavailable'
        envelope['status'] = 'partial' if status != 'refused' else 'refused'
        return 'report'

    if envelope['status'] != 'partial':
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
