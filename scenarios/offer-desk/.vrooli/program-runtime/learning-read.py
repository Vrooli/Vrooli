"""Read Offer Desk task outcomes over explicit comparable windows; never infer a missing cohort."""
import json

inputs = program.inputs()

envelope = {'program': 'offer-desk.learning-read', 'version': '1', 'status': 'failed',
            'phase': 'validate', 'inputs': {}, 'signals': {}, 'errors': [], 'evidence': []}

SCOPE = 'offer-desk-usage'
COHORT_CAP = 12


guarded = program.guarded


fail = program.fail


state = {'operation': '', 'handle': None}


def step_validate():
    # phase: validate — inputs only; no binding may be called here.
    operation = inputs.get('operation') or ''
    if not isinstance(operation, str):
        return fail('failed', 'invalid_input', 'operation must be a string', 'validate')
    state['operation'] = operation.strip()
    envelope['inputs'] = {'scope': SCOPE, 'operation': state['operation']}
    return 'collect'


def step_collect():
    # phase: collect — one governed read against the declared usage scope.
    envelope['phase'] = 'collect'
    if state['operation']:
        call = lambda: vrooli_memory.learning.measure(scope=SCOPE, operation=state['operation'], rows='cohorts')
    else:
        call = lambda: vrooli_memory.learning.measure(scope=SCOPE, rows='cohorts')
    result = gather(guarded(call))[0]
    if isinstance(result, Exception):
        status, klass = program.classify(result)
        return fail(status, klass, result, 'collect')
    state['handle'] = result
    envelope['evidence'].append('vrooli-memory/learning/measure')
    return 'classify'


def step_classify():
    # phase: classify — copy the producer's own validity verdict; never recompute reliability.
    envelope['phase'] = 'classify'
    handle = state['handle']
    meta = handle.meta()
    # protojson omits an int at zero and false, so absence here is a real zero, not unknown.
    eligible = int(meta.get('eligibleAttempts', 0) or 0)
    reliable = bool(meta.get('reliable', False))
    truncated = bool(meta.get('truncated', False))
    reason = meta.get('reason')
    cohorts = handle.head(COHORT_CAP)

    envelope['signals'] = {
        'scope': SCOPE,
        'operation': state['operation'] or None,
        'eligible_attempts': eligible,
        'cohort_count': handle.count(),
        'cohorts': cohorts,
        'reliable': reliable,
        'truncated': truncated,
        'producer_reason': reason,
        # A scope with no attempts is a real, correct reading of "nothing learned yet".
        # It is never presented as a measured comparison.
        'comparable': reliable and not truncated and eligible > 0,
        'comparability_reason': None if (reliable and not truncated and eligible > 0)
        else (str(reason) if reason else ('truncated' if truncated else 'no_eligible_attempts')),
    }
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
