"""Read target and plan catalogs together through their typed owner bindings."""
import json

inputs = program.inputs()
envelope = program.envelope('data-backup-manager.catalog-read', '1', phase='validate', inputs={})
guarded = program.guarded
fail = program.fail
state = {'owner': '', 'targets': [], 'plans': [], 'target_token': '', 'plan_token': '', 'truncated': False}
PAGE_SIZE = 50
MAX_PAGES = 4


def step_validate():
    owner = inputs.get('owner', '')
    if not isinstance(owner, str):
        return fail('failed', 'invalid_input', 'owner must be a string', 'validate')
    state['owner'] = owner.strip()
    envelope['inputs'] = {'owner_filter': state['owner']}
    return 'collect'


def step_collect():
    envelope['phase'] = 'collect'
    def collect_pages(binding, owner=None):
        rows, token = [], ''
        for _ in range(MAX_PAGES):
            kwargs = {'page-size': PAGE_SIZE, 'page-token': token}
            if owner is not None:
                kwargs['owner'] = owner
            page = binding(**kwargs)
            rows.extend(page.head(PAGE_SIZE))
            token = page.meta().get('nextPageToken', '') or ''
            if not token:
                return rows, ''
        return rows, token

    calls = [lambda: collect_pages(data_backup_manager.targets.list, state['owner']),
             lambda: collect_pages(data_backup_manager.plans.list)]
    results = gather(guarded(calls[0]), guarded(calls[1]))
    if not isinstance(results[0], Exception):
        try:
            target_rows, state['target_token'] = results[0]
            state['targets'] = []
            for row in target_rows:
                state['targets'].append({key: row.get(key) for key in ('id', 'owner', 'name', 'sourceKind', 'locator', 'critical') if key in row})
            envelope['evidence'].append('data-backup-manager/targets/list')
        except Exception as exc:
            results[0] = exc
    if not isinstance(results[1], Exception):
        try:
            plan_rows, state['plan_token'] = results[1]
            state['plans'] = []
            for row in plan_rows:
                state['plans'].append({
                    'id': row.get('id'), 'name': row.get('name'), 'enabled': row.get('enabled'),
                    'schedule': row.get('schedule'), 'recoveryDrillSchedule': row.get('recoveryDrillSchedule'),
                    'protectionTier': row.get('protectionTier'), 'targetSelector': row.get('targetSelector'),
                    'targetCount': len(row.get('targetIds') or []), 'destinationIds': row.get('destinationIds') or [],
                })
            envelope['evidence'].append('data-backup-manager/plans/list')
        except Exception as exc:
            results[1] = exc
    errors = []
    for label, result in [('targets', results[0]), ('plans', results[1])]:
        if isinstance(result, Exception):
            status, klass = program.classify(result)
            errors.append({'class': klass, 'detail': str(result), 'where': label})
    if errors:
        envelope['errors'].extend(errors)
        envelope['status'] = 'partial' if len(errors) == 1 else 'unavailable'
    elif state['target_token'] or state['plan_token'] or state['truncated']:
        envelope['status'] = 'partial'
        envelope['errors'].append({'class': 'row_cap_reached', 'detail': 'The bounded four-page scan has a continuation token; use the owner list operation with that token to continue.', 'where': 'collect'})
    else:
        envelope['status'] = 'ok'
    return 'report'


def step_report():
    envelope['phase'] = 'report'
    envelope['signals'] = {
        'owner_filter': state['owner'],
        'targets': state['targets'],
        'plans': state['plans'],
        'targets_next_page_token': state['target_token'],
        'plans_next_page_token': state['plan_token'],
        'truncated': state['truncated'] or bool(state['target_token']) or bool(state['plan_token']),
    }
    print(json.dumps(envelope, allow_nan=False, separators=(',', ':')))
    return None


program.run({'validate': step_validate, 'collect': step_collect, 'report': step_report}, 'validate')
