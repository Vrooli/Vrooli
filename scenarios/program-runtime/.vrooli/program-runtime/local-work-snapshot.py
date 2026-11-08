"""One read-only cross-owner observation. Never infer work acceptance or liveness."""
import datetime
import hashlib
import json
import os

inputs = program.inputs()
envelope = program.envelope('program-runtime.local-work-snapshot', '2')
STATES = ('pending', 'starting', 'running', 'needs_review', 'parked', 'unknown', 'complete', 'failed', 'cancelled')
SAMPLES = ('teams', 'objectives', 'swarm_goals', 'research_topics', 'research_runs', 'program_usage')
DEFAULTS = {'research_root': '', 'row_limit': 4, 'run_statuses': list(STATES[:7]), 'run_offsets': {}, 'sample_offsets': {}, 'effort_page_token': '', 'usage_window_days': 7, 'team_ids': [], 'team_task_offsets': {}}
config = {}


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def integer(value):
    if isinstance(value, bool):
        raise ValueError('boolean is not a count')
    if isinstance(value, str) and value.isdigit() and len(value) <= 19:
        value = int(value)
    if not isinstance(value, int) or not 0 <= value <= 9223372036854775807:
        raise ValueError('missing, null or invalid count')
    return value


def bounded(value, depth=0):
    """Projection text is capped, never silently treated as complete."""
    if isinstance(value, str):
        return value if len(value) <= 240 else value[:240] + ' [text truncated]'
    if isinstance(value, dict):
        if depth >= 4:
            return {'omitted': 'nested detail; inspect owner'}
        result = {k: bounded(v, depth + 1) for k, v in list(value.items())[:12]}
        if len(value) > 12:
            result['omitted_fields'] = len(value) - 12
        return result
    if isinstance(value, list):
        result = [bounded(v, depth + 1) for v in value[:8]]
        if len(value) > 8:
            result.append({'omitted_items': len(value) - 8})
        return result
    return value


def pick(row, keys):
    if not isinstance(row, dict):
        raise ValueError('owner row is not an object')
    return {key: bounded(row[key]) for key in keys if key in row}


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, allow_nan=False).encode()).hexdigest()


def source(name, locator, scope):
    item = {'owner_locator': locator, 'queried_scope': scope, 'collected_at': now(), 'producer_observed_at': None,
            'freshness': 'unknown; fresh retrieval does not freshen producer claims', 'status': 'unknown',
            'response_rows': None, 'returned_rows': 0, 'query_complete': None, 'entities': [], 'errors': []}
    envelope['signals']['sources'][name] = item
    return item


def failed(item, exc):
    if isinstance(exc, (NameError, AttributeError)):
        raise exc
    item['status'] = 'unavailable' if isinstance(exc, OSError) else 'unknown'
    item['query_complete'] = False
    detail = str(exc)[:240]
    klass = 'invalid_response' if isinstance(exc, (ValueError, TypeError)) else 'source_unavailable'
    item['errors'].append({'class': klass, 'detail': detail})
    envelope['errors'].append({'class': klass, 'detail': item['owner_locator'] + ': ' + detail, 'where': 'collect'})


def rows_read(name, locator, scope, call, project, offset=0, page=False):
    item = source(name, locator, scope)
    try:
        handle = call()
        count = integer(handle.count())
        meta = handle.meta()
        if not isinstance(meta, dict):
            raise ValueError('owner metadata is not an object')
        item['response_rows'] = count
        take = min(count, offset + config['row_limit'], 200)
        rows = handle.head(take)
        if not isinstance(rows, list) or len(rows) != take:
            raise ValueError('owner handle count and rows disagree')
        selected = rows[offset:offset + config['row_limit']]
        item['entities'] = [project(row) for row in selected]
        item['returned_rows'] = len(selected)
        item['sample_offset'] = offset
        item['next_sample_offset'] = offset + len(selected) if offset + len(selected) < min(count, 200) else None
        item['inspection_bound_reached'] = count > 200 and offset + len(selected) >= 200
        item['output_complete'] = offset == 0 and len(selected) == count
        item['query_complete'] = not page
        item['status'] = 'ok'
        item['selected_projection_sha256'] = digest(item['entities'])
        return item, meta, handle
    except Exception as exc:
        failed(item, exc)
        return item, None, None


# validate: invalid inputs must perform no owner reads.
def step_validate():
    global config
    config = {key: inputs.get(key, default) for key, default in DEFAULTS.items()}
    for key, low, high in (('row_limit', 1, 20), ('usage_window_days', 1, 30)):
        value = config[key]
        if isinstance(value, bool) or not isinstance(value, int) or not low <= value <= high:
            return program.fail('failed', 'invalid_input', key + ' outside allowed range', 'validate')
    statuses = config['run_statuses']
    if not isinstance(statuses, list) or not statuses or len(statuses) > 9 or any(not isinstance(x, str) or x not in STATES for x in statuses) or len(set(statuses)) != len(statuses):
        return program.fail('failed', 'invalid_input', 'invalid selected run statuses', 'validate')
    for key, allowed, maximum in (('run_offsets', statuses, 1000000), ('sample_offsets', SAMPLES, 199)):
        values = config[key]
        if not isinstance(values, dict) or any(k not in allowed or isinstance(v, bool) or not isinstance(v, int) or not 0 <= v <= maximum for k, v in values.items()):
            return program.fail('failed', 'invalid_input', 'invalid ' + key, 'validate')
    team_ids = config['team_ids']
    if not isinstance(team_ids, list) or len(team_ids) > 4 or any(not isinstance(x, str) or not x or len(x) > 128 or any(c not in 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_' for c in x) for x in team_ids) or len(set(team_ids)) != len(team_ids):
        return program.fail('failed', 'invalid_input', 'team_ids must contain up to four unique owner IDs', 'validate')
    offsets = config['team_task_offsets']
    if not isinstance(offsets, dict) or any(k not in team_ids or isinstance(v, bool) or not isinstance(v, int) or not 0 <= v <= 1000000 for k, v in offsets.items()):
        return program.fail('failed', 'invalid_input', 'invalid team_task_offsets', 'validate')
    root = config['research_root']
    token = config['effort_page_token']
    if not isinstance(root, str) or len(root) > 1024 or '\0' in root or (root and not os.path.isabs(root)) or not isinstance(token, str) or len(token) > 4096:
        return program.fail('failed', 'invalid_input', 'invalid root or effort continuation', 'validate')
    if root:
        config['research_root'] = os.path.realpath(root)
    envelope['inputs'] = config
    envelope['signals'] = {'collected_at': now(), 'sources': {}, 'output_truncated': False,
        'coverage': {'kind': 'bounded_observation', 'selected_team_ids': team_ids, 'unselected_run_statuses': [s for s in STATES if s not in statuses],
                     'not_collected': ['native process liveness', 'team queues and task boards outside explicitly selected team_ids', 'Swarm execution records', 'research scientific acceptance', 'Flywheel ownership and matches']},
        'reconciliation': {'decision': 'review_required', 'relations': [],
            'allowed_relations': ['contributes_to', 'depends_on', 'duplicates', 'validates', 'informs', 'follow_up_candidate', 'local_only', 'cloud_only'],
            'required_per_relation': ['source-native identities', 'source evidence and time', 'confidence', 'owning outcome and reviewer'],
            'rule': 'No match is not missing work. No label, self-report or runtime success closes an outcome.'}}
    return 'collect'


def run_row(row):
    result = pick(row, ('id', 'label', 'status', 'executionMode', 'taskId', 'parentRunId', 'updatedAt', 'startedAt', 'endedAt', 'harnessKind', 'harnessSessionId', 'importSourceHarness', 'importSourceSessionId'))
    result.update(entity_type='agent_run', execution_liveness='unverified', accepted_outcome='unverified')
    return result


def team_row(row):
    result = pick(row, ('id', 'displayName', 'purpose', 'lifetime', 'effortRefs', 'objectivesServed', 'homePath', 'updatedAt'))
    # These are non-optional protobuf scalars: omitted means false, explicit null is invalid.
    for key in ('enabled', 'archived'):
        value = row.get(key, False)
        if not isinstance(value, bool):
            raise ValueError('invalid team ' + key)
        result[key] = value
    result['entity_type'] = 'team_container'
    result['execution_liveness'] = 'not_checked'
    return result


def effort_row(row):
    result = pick(row, ('observedAt', 'freshness', 'runtimeState', 'nextAction'))
    enrollment = row.get('enrollment')
    result['enrollment'] = pick(enrollment, ('effortRef', 'displayName', 'destinationRef', 'workspace', 'workShape', 'updatedAt', 'withdrawn'))
    # Keep identity exact, while prose and supporting evidence stay bounded.
    for key in ('effortRef', 'destinationRef', 'workspace'):
        if key in enrollment:
            result['enrollment'][key] = enrollment[key]
    subjects = enrollment.get('subjects', [])
    if not isinstance(subjects, list):
        raise ValueError('invalid effort subjects')
    result['subjects'] = [pick(r, ('owner', 'kind', 'reference', 'runId', 'role')) for r in subjects[:3]]
    result['subjects_omitted'] = max(0, len(subjects) - 3)
    result['outcomeStanding'] = pick(row.get('outcomeStanding', {}), ('state', 'attribution', 'metCount', 'requiredCount'))
    for key in ('limitations', 'evidenceRefs'):
        values = row.get(key, [])
        if not isinstance(values, list):
            raise ValueError('invalid effort ' + key)
        result[key] = bounded(values[:2])
        result[key + '_omitted'] = max(0, len(values) - 2)
    result['entity_type'] = 'enrolled_effort'
    return result


def goal_row(row):
    result = pick(row.get('goal'), ('name', 'title', 'status', 'targets', 'servesDeliverable', 'updated', 'archivedAt'))
    result.update(entity_type='swarm_goal', execution_liveness='not_checked', accepted_outcome='unverified')
    return result


# Queue ownership is an owner projection, not a process-liveness oracle.
TEAM_CLASSES = ('runningAgentIds', 'queue', 'uncertainAgentIds', 'unreachableAgentIds',
                'pausedAgentIds', 'approvalPendingAgentIds', 'needsReviewAgentIds',
                'watcherUnavailableAgentIds', 'preparationInterruptedAgentIds', 'admissionDeferredAgentIds')


def exact_identity(value):
    if not isinstance(value, str) or len(value) > 1024 or '\0' in value:
        raise ValueError('invalid owner identity')
    return value


def team_data(handle, team_id):
    meta = handle.meta()
    data = meta.get('data') if isinstance(meta, dict) else None
    if not isinstance(data, dict) or data.get('teamId') != team_id:
        raise ValueError('missing or mismatched selected team identity')
    return data


def team_owner_row(row):
    result = pick(row, ('agentId', 'phase', 'state', 'dispatchStage', 'dispatchFailurePhase',
                        'dispatchFailureCode', 'taskId', 'runId', 'admissionOrigin', 'ownerStartedAt', 'obligationId'))
    if not row.get('agentId') or row.get('phase') not in ('retained', 'queued'):
        raise ValueError('invalid retained or queued owner')
    for key in ('agentId', 'taskId', 'runId', 'obligationId'):
        if key in row:
            result[key] = exact_identity(row[key])
    result.update(entity_type='team_execution_owner', execution_liveness='unverified', accepted_outcome='unverified')
    return result


def team_task_row(row):
    result = pick(row, ('id', 'title', 'status', 'assignee', 'priority', 'createdBy', 'createdAt', 'updatedAt'))
    if not row.get('id'):
        raise ValueError('missing team task identity')
    for key in ('id', 'assignee', 'createdBy'):
        if key in row:
            result[key] = exact_identity(row[key])
    result.update(entity_type='team_board_task', execution_liveness='unverified', accepted_outcome='unverified')
    return result


def teams_read():
    if not config['team_ids']:
        source('team_activity', 'prompt-manager/team/execution-status + prompt-manager/team/task-list',
               'no team_ids selected')['status'] = 'not_requested'
        return
    limit = config['row_limit']
    for team_id in config['team_ids']:
        item = source('team_execution/' + team_id, 'prompt-manager/team/execution-status', {'team_id': team_id})
        try:
            data = team_data(prompt_manager.team.execution_status(team_id=team_id), team_id)
            if data.get('state') not in ('idle', 'active'):
                raise ValueError('invalid team execution state')
            owners = data.get('owners')
            if not isinstance(owners, list):
                raise ValueError('missing or invalid team owner list')
            counts = {}
            for key in TEAM_CLASSES:
                values = data.get(key)
                # JsonResponse.data is an untyped Struct. Missing or null fields
                # here have no protobuf default-zero/empty-array guarantee.
                if not isinstance(values, list) or any(not isinstance(x, str) or not x for x in values):
                    raise ValueError('missing or invalid team execution ' + key)
                counts[key] = len(values)
            item.update(status='ok', team_id=team_id, owner_state=data['state'],
                        queue_policy=bounded(data.get('queuePolicy')), max_concurrent_runs=integer(data.get('maxConcurrentRuns')),
                        response_rows=len(owners), returned_rows=min(limit, len(owners)), query_complete=True,
                        output_complete=len(owners) <= limit and len(data['queue']) <= limit, entities=[team_owner_row(row) for row in owners[:limit]],
                        owner_class_counts=counts, owners_omitted=max(0, len(owners) - limit),
                        queue_sample=[exact_identity(x) for x in data['queue'][:limit]], queue_members_omitted=max(0, len(data['queue']) - limit))
            item['selected_projection_sha256'] = digest({'entities': item['entities'], 'queue_sample': item['queue_sample'], 'owner_class_counts': counts})
            item['note'] = 'Owner-reported retained slots and queue only. Class counts overlap and must not be summed. No producer observation time is supplied. Idle may also be the owner fallback when its execution store is absent; it is not proof of no work or live process state.'
        except Exception as exc:
            failed(item, exc)
        offset = config['team_task_offsets'].get(team_id, 0)
        item = source('team_tasks/' + team_id, 'prompt-manager/team/task-list', {'team_id': team_id, 'limit': limit, 'offset': offset, 'order': 'owner-defined'})
        try:
            data = team_data(prompt_manager.team.task_list(team_id=team_id, query={'limit': str(limit), 'offset': str(offset)}), team_id)
            tasks = data.get('tasks')
            total = integer(data.get('total'))
            if (integer(data.get('limit')) != limit or integer(data.get('offset')) != offset
                    or not isinstance(tasks, list) or len(tasks) != min(limit, max(0, total - offset))):
                raise ValueError('inconsistent team task pagination')
            item.update(status='ok', team_id=team_id, response_rows=len(tasks), returned_rows=len(tasks),
                        total_in_scope=total, query_complete=offset == 0 and len(tasks) == total,
                        output_complete=True, next_offset=offset + len(tasks) if offset + len(tasks) < total else None,
                        entities=[team_task_row(row) for row in tasks])
            item['selected_projection_sha256'] = digest(item['entities'])
            item['note'] = 'One retained task-board page. Empty does not mean no team work; done does not prove accepted outcome. Task updatedAt is producer evidence, not refreshed by this read.'
        except Exception as exc:
            failed(item, exc)


def research_read():
    root = config['research_root']
    if not root:
        source('research_catalog', 'CATALOG.json', 'not requested')['status'] = 'not_requested'
        source('research_workspace', 'program-runtime.research-workspace-review', 'not requested')['status'] = 'not_requested'
        return
    path = os.path.join(root, 'CATALOG.json')
    item = source('research_catalog', path, 'one catalog file; retained entries, not live commissions')
    try:
        if os.path.commonpath((root, os.path.realpath(path))) != root:
            raise ValueError('catalog resolves outside authorized root')
        with open(path, 'rb') as stream:
            data = stream.read(1048577)
        if len(data) > 1048576:
            raise ValueError('catalog exceeds 1 MiB read bound')
        catalog = json.loads(data)
        if not isinstance(catalog, dict):
            raise ValueError('catalog is not an object')
        item['producer_observed_at'] = catalog.get('updated_at')
        item['file_mtime'] = datetime.datetime.fromtimestamp(os.stat(path).st_mtime, datetime.timezone.utc).isoformat()
        item['content_sha256'] = hashlib.sha256(data).hexdigest()
        item['catalog_version'] = catalog.get('version')
        item['status'] = 'ok'
        item['query_complete'] = True
        item['summary'] = {}
        for name, keys in (('topics', ('id', 'title', 'guide', 'selected_workspace')), ('runs', ('id', 'topic', 'kind', 'title', 'workspace', 'reading_guide', 'summary'))):
            values = catalog.get(name)
            if not isinstance(values, list):
                raise ValueError('missing or invalid catalog ' + name)
            offset = config['sample_offsets'].get('research_' + name, 0)
            item['summary'][name + '_retained_entries'] = len(values)
            item[name] = [pick(row, keys) for row in values[offset:offset + config['row_limit']]]
            item[name + '_next_sample_offset'] = offset + len(item[name]) if offset + len(item[name]) < len(values) else None
        item['note'] = 'Catalog paths are locators only; not followed. Resolve aliases and parent/support roles before joining. Directory counts have a different denominator.'
    except Exception as exc:
        failed(item, exc)
    child_item = source('research_workspace', 'program-runtime.research-workspace-review', {'research_root': root})
    try:
        h = lib.program_runtime.research_workspace_review(research_root=root)
        child = program.child_envelope(h, invalid_class='invalid_response', where='collect:research_workspace')
        child_item['status'] = child['status']
        child_item['query_complete'] = child['status'] == 'ok'
        child_item['signals'] = bounded(child['signals'])
        child_item['errors'] = bounded(child['errors'])
        child_item['artifact'] = pick(h.meta(), ('program', 'version', 'digest'))
        child_item['note'] = 'Filesystem presence and sizes only. No finding, commission status or acceptance is inferred.'
    except Exception as exc:
        failed(child_item, exc)


# collect: each selected source is read once; no retry or polling.
def step_collect():
    envelope['phase'] = 'collect'
    limit = config['row_limit']
    for status in config['run_statuses']:
        offset = config['run_offsets'].get(status, 0)
        item, meta, handle = rows_read('runs/' + status, 'agent-manager/run/list', {'status': status, 'limit': limit, 'offset': offset},
            lambda s=status, o=offset: agent_manager.run.list(status='RUN_STATUS_' + s.upper(), limit=limit, offset=o), run_row, page=True)
        if meta is not None:
            try:
                total = integer(meta.get('total', 0))
                more = meta.get('hasMore', False)
                if not isinstance(more, bool) or total < item['response_rows'] or (not more and offset + item['response_rows'] < total):
                    raise ValueError('inconsistent owner pagination')
                item.update(total_in_scope=total, has_more=more, next_offset=offset + item['response_rows'] if more else None,
                            query_complete=offset == 0 and not more, note='Status is retained owner state; imported/attached running records do not prove live executors.')
            except Exception as exc:
                failed(item, exc)
    item, meta, handle = rows_read('teams', 'prompt-manager/team/list', 'registered team containers; up to 200 examined for lifetime summary', lambda: prompt_manager.team.list(), team_row, config['sample_offsets'].get('teams', 0))
    if handle is not None:
        counts = {}
        for row in handle.head(200):
            key = row.get('lifetime', 'unspecified')
            counts[key] = counts.get(key, 0) + 1
        item['lifetime_counts'] = counts
        item['summary_complete'] = handle.count() <= 200
        item['note'] = 'Standing objective teams and finite delivery containers are not executors; task queues are separate.'
    teams_read()
    item = source('objectives', 'prompt-manager/graph/objectives', 'declared objective attachments, not achieved outcomes')
    try:
        h = prompt_manager.graph.objectives()
        data = h.meta().get('data')
        if not isinstance(data, dict) or not isinstance(data.get('rows'), list):
            raise ValueError('invalid objective projection')
        rows = data['rows']
        offset = config['sample_offsets'].get('objectives', 0)
        item.update(status='ok', response_rows=len(rows), returned_rows=len(rows[offset:offset + limit]), query_complete=True,
            entities=[pick(r, ('id', 'title', 'class', 'meaningRevision', 'served', 'servedBy', 'evidenceSource')) for r in rows[offset:offset + limit]],
            source_path=data.get('sourcePath'), next_sample_offset=offset + limit if offset + limit < len(rows) else None,
            validation=bounded(data.get('validation')), drift=bounded(data.get('drift')))
    except Exception as exc:
        failed(item, exc)
    item, meta, handle = rows_read('efforts', 'agent-manager/effort/board', {'page_size': limit, 'page_token': config['effort_page_token']},
        lambda: agent_manager.effort.board(page_size=limit, page_token=config['effort_page_token'], rows='rows'), effort_row, page=True)
    if meta is not None:
        item['producer_observed_at'] = meta.get('observedAt')
        item['owner_metadata'] = pick(meta, ('activeCount', 'partial', 'nextPageToken', 'changeIdentity', 'limitations'))
        item['discovery'] = pick(meta.get('discovery', {}), ('lastScanAt', 'lastSuccessfulScanAt', 'partial', 'scannedCount', 'scanLimit', 'root', 'findings', 'scanCursor'))
        item['query_complete'] = not config['effort_page_token'] and not meta.get('nextPageToken') and meta.get('partial', False) is False
        item['note'] = 'A complete page does not imply complete discovery. activeCount is not live executor count.'
    item, meta, handle = rows_read('swarm_goals', 'swarm-manager/goals/list', 'all returned goal declarations, including archived', lambda: swarm_manager.goals.list(), goal_row, config['sample_offsets'].get('swarm_goals', 0))
    if handle is not None:
        counts = {}
        for row in handle.head(200):
            key = row.get('goal', {}).get('status', 'unspecified')
            counts[key] = counts.get(key, 0) + 1
        item['status_counts'] = counts
        item['summary_complete'] = handle.count() <= 200
        # Focus the useful examples without hiding archived goal counts.
        candidates = handle.head(200)
        candidates.sort(key=lambda row: row.get('goal', {}).get('updated', ''), reverse=True)
        candidates.sort(key=lambda row: row.get('goal', {}).get('status') != 'active')
        offset = config['sample_offsets'].get('swarm_goals', 0)
        item['entities'] = [goal_row(row) for row in candidates[offset:offset + limit]]
        item['selected_projection_sha256'] = digest(item['entities'])
        item['example_order'] = 'active declarations first, then updated descending within the inspected first 200 owner rows'

    item, meta, handle = rows_read('swarm_backlog', 'swarm-manager/backlog/list', {'limit': limit, 'order': 'owner-defined; not necessarily active or newest'},
        lambda: swarm_manager.backlog.list(limit=limit), lambda row: pick(row, ('kind', 'name', 'title', 'status', 'updated', 'planRef', 'stale')), page=True)
    if meta is not None:
        try:
            item['total_in_scope'] = integer(meta.get('total', 0))
            item['query_complete'] = item['total_in_scope'] == item['response_rows']
            item['continuation'] = 'No cursor advertised by this binding; narrow through owner CLI, do not infer absence from this sample.'
        except Exception as exc:
            failed(item, exc)
    gap = source('swarm_executions', 'swarm-manager execution list --json', 'not collected by this program')
    gap.update(status='not_collected', query_complete=False, reason='No execution list binding in the inspected governed catalog. Use the supported owner CLI separately; no shell fallback inside a program.')
    research_read()
    item, meta, handle = rows_read('program_usage', 'program-runtime/programs/portfolio', {'window_days': config['usage_window_days']},
        lambda: program_runtime.programs.portfolio(window_days=config['usage_window_days']),
        lambda row: pick(row, ('name', 'scenario', 'currentDigest', 'runs', 'succeeded', 'failed', 'firstSeen', 'lastSeen')),
        config['sample_offsets'].get('program_usage', 0))
    if meta is not None:
        item['window'] = pick(meta, ('windowStart', 'windowEnd', 'programsDeclared', 'programsExecuted', 'rowsWithoutIdentity', 'unattributedAgentRuns', 'contractIndexReason'))
        item['note'] = 'Capability and usage context only; runtime success does not establish domain outcome success.'
    envelope['status'] = 'partial'
    return 'report'


# report: reduce oversized entity examples explicitly; retain source coverage.
def step_report():
    envelope['phase'] = 'report'
    if 'sources' in envelope['signals']:
        envelope['signals']['finished_at'] = now()
        sources = envelope['signals']['sources']
        for name in sorted(sources, key=lambda key: len(json.dumps(sources[key])), reverse=True):
            if len(json.dumps(envelope, allow_nan=False).encode()) <= 60000:
                break
            item = sources[name]
            omitted = len(item.get('entities', []))
            item['entities'] = []
            item['returned_rows'] = 0
            item['output_complete'] = False
            item['omitted_due_to_output_limit'] = omitted
            item['status_before_output_limit'] = item['status']
            item['status'] = 'partial'
            envelope['signals']['output_truncated'] = True
        # A pathological owner projection can exceed the tier even after rows
        # are removed. Preserve coverage and identity, and explicitly omit details.
        for name in sorted(sources, key=lambda key: len(json.dumps(sources[key])), reverse=True):
            if len(json.dumps(envelope, allow_nan=False).encode()) <= 60000:
                break
            item = sources[name]
            removed = []
            for key in ('topics', 'runs', 'signals', 'validation', 'drift', 'discovery', 'queue_sample'):
                if key in item:
                    del item[key]
                    removed.append(key)
            if removed:
                item['details_omitted_due_to_output_limit'] = removed
                item['output_complete'] = False
                item['status'] = 'partial'
                envelope['signals']['output_truncated'] = True
        if envelope['signals']['output_truncated']:
            envelope['errors'].append({'class': 'output_limit', 'detail': 'Entity examples omitted; rerun with smaller row_limit or selected status set.', 'where': 'report'})
    print(json.dumps(envelope, allow_nan=False, separators=(',', ':')))
    return None


program.run({'validate': step_validate, 'collect': step_collect, 'report': step_report}, 'validate')
