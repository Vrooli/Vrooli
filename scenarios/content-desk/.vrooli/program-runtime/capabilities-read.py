"""Read the marketing capability catalog and report readiness gaps, limitations and next actions."""
import json

inputs = program.inputs()

envelope = {'program': 'content-desk.capabilities-read', 'version': '1', 'status': 'failed',
            'phase': 'validate', 'inputs': {}, 'signals': {}, 'errors': [], 'evidence': []}

ROW_CAP = 80
SAMPLE_CAP = 8
READINESS_DIMENSIONS = ('definition_status', 'implementation_status', 'operational_readiness',
                        'output_quality', 'distribution_connectivity')

guarded = program.guarded
fail = program.fail

state = {}


def _str(value):
    return '' if value is None else str(value)


def _field(row, camel, snake):
    # protojson returns camelCase; keep the snake_case fallback for a manual caller.
    value = row.get(camel)
    if value is None:
        value = row.get(snake)
    return value


def _dedupe(values, cap=SAMPLE_CAP):
    seen = []
    for value in values:
        text = _str(value).strip()
        if text and text not in seen:
            seen.append(text)
        if len(seen) >= cap:
            break
    return seen


def _looks_like_uuid(value):
    if len(value) != 36:
        return False
    parts = value.split('-')
    if [len(p) for p in parts] != [8, 4, 4, 4, 12]:
        return False
    return all(all(c in '0123456789abcdefABCDEF' for c in p) for p in parts)


def step_validate():
    # phase: validate — inputs only; no binding may be called here.
    medium = _str(inputs.get('medium')).strip()
    capability = _str(inputs.get('capability')).strip()
    state['medium'] = medium
    state['capability'] = capability
    envelope['inputs'] = {'medium': medium, 'capability': capability}
    return 'collect'


def step_collect():
    # phase: collect — governed reads; the catalog is the single declared address.
    envelope['phase'] = 'collect'

    list_kwargs = {}
    if state['medium']:
        list_kwargs['medium'] = state['medium']
    reads = {'list': guarded(lambda: content_desk.capabilities.list(**list_kwargs))}
    if state['capability']:
        if _looks_like_uuid(state['capability']):
            get_kwargs = {'id': state['capability']}
        else:
            get_kwargs = {'alias': state['capability']}
        reads['get'] = guarded(lambda: content_desk.capabilities.get(**get_kwargs))

    names = list(reads.keys())
    outcomes = gather(*[reads[name] for name in names])
    by_name = dict(zip(names, outcomes))

    board = by_name['list']
    if isinstance(board, Exception):
        status, klass = program.classify(board)
        return fail(status, klass, board, 'collect')
    try:
        state['total'] = board.count()
        state['rows'] = board.head(ROW_CAP)
        state['truncated'] = state['total'] > ROW_CAP
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, 'collect')
    envelope['evidence'].append('content-desk/capabilities/list')

    if 'get' in by_name:
        focus = by_name['get']
        if isinstance(focus, Exception):
            http = getattr(focus, 'http_status', 0)
            if http == 404:
                # An absent capability is a reported state, not a failed read.
                state['focus_found'] = False
                state['focus'] = None
            else:
                status, klass = program.classify(focus)
                return fail(status, klass, focus, 'collect')
        else:
            meta = focus.meta() or {}
            capability = meta.get('capability') or meta.get('Capability')
            if not capability:
                head = focus.head(1)
                capability = head[0] if head else None
            state['focus'] = capability
            state['focus_found'] = bool(capability)
            if capability:
                envelope['evidence'].append(_str(capability.get('id')))
        envelope['evidence'].append('content-desk/capabilities/get')
    return 'classify'


def step_classify():
    # phase: classify — a deterministic table; no judgment is needed to count owner records.
    envelope['phase'] = 'classify'
    rows = state.get('rows') or []

    by_medium = {}
    readiness = {dim: {} for dim in READINESS_DIMENSIONS}
    unknown_qualification = 0
    ownerless = 0
    with_limitations = 0
    limitations = []
    next_actions = []
    for row in rows:
        medium = _str(_field(row, 'medium', 'medium'))
        by_medium[medium] = by_medium.get(medium, 0) + 1
        for dim in READINESS_DIMENSIONS:
            camel = ''.join(part.title() for part in dim.split('_'))
            camel = camel[0].lower() + camel[1:]
            value = _str(_field(row, camel, dim))
            readiness[dim][value] = readiness[dim].get(value, 0) + 1
        qualification = _field(row, 'latestQualification', 'latest_qualification')
        observed_at = _str(_field(qualification or {}, 'observedAt', 'observed_at')).strip().lower()
        try:
            max_age = int(_field(qualification or {}, 'maxAgeSeconds', 'max_age_seconds') or -1)
        except (TypeError, ValueError):
            max_age = -1
        if not qualification or observed_at in ('', 'unknown') or max_age < 0:
            unknown_qualification += 1
        if not _str(_field(row, 'owner', 'owner')):
            ownerless += 1
        row_limits = _field(row, 'readinessLimitations', 'readiness_limitations') or []
        if row_limits:
            with_limitations += 1
            limitations.extend(row_limits)
        action = _str(_field(row, 'nextAction', 'next_action'))
        if action:
            next_actions.append(action)

    focus_signal = None
    if state.get('capability'):
        capability = state.get('focus')
        if capability:
            focus_signal = {
                'id': _str(capability.get('id')),
                'name': _str(capability.get('name')),
                'medium': _str(capability.get('medium')),
                'owner': _str(capability.get('owner')),
                'priority': capability.get('priority'),
                'operational_readiness': _str(_field(capability, 'operationalReadiness', 'operational_readiness')),
                'output_quality': _str(_field(capability, 'outputQuality', 'output_quality')),
                'distribution_connectivity': _str(_field(capability, 'distributionConnectivity', 'distribution_connectivity')),
                'next_action': _str(_field(capability, 'nextAction', 'next_action')),
                'limitations': _dedupe(_field(capability, 'readinessLimitations', 'readiness_limitations') or []),
            }

    envelope['signals'] = {
        'total': state.get('total', len(rows)),
        'materialized': len(rows),
        'truncated': bool(state.get('truncated')),
        'by_medium': by_medium,
        'readiness': readiness,
        'unknown_qualification': unknown_qualification,
        'ownerless': ownerless,
        'with_limitations': with_limitations,
        'limitations': _dedupe(limitations, 16),
        'next_actions': _dedupe(next_actions, 16),
        'focus_found': state.get('focus_found') if state.get('capability') else None,
        'focus': focus_signal,
    }

    if state.get('truncated'):
        envelope['status'] = 'partial'
        envelope['errors'].append({'class': 'row_cap_reached',
                                   'detail': 'catalog returned more capabilities than the materialize cap; report covers the first %d' % ROW_CAP,
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
program.run(STATES, 'validate')
