"""Cross-owner observations must never turn bounded/stale evidence into completion."""
import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace

SOURCE = Path(__file__).parents[1] / 'local-work-snapshot.py'
TEAM_CLASSES = ('runningAgentIds', 'queue', 'uncertainAgentIds', 'unreachableAgentIds',
                'pausedAgentIds', 'approvalPendingAgentIds', 'needsReviewAgentIds',
                'watcherUnavailableAgentIds', 'preparationInterruptedAgentIds', 'admissionDeferredAgentIds')


class Handle:
    def __init__(self, rows=None, meta=None):
        self.rows = [] if rows is None else rows
        self.metadata = {} if meta is None else meta
    def count(self):
        return len(self.rows)
    def head(self, count):
        return self.rows[:count]
    def meta(self):
        return self.metadata


def execute(inputs=None, overrides=None):
    state, calls = {}, []
    inputs, overrides = inputs or {}, overrides or {}
    def read(name, **kwargs):
        calls.append((name, kwargs))
        value = overrides.get(name)
        if isinstance(value, Exception):
            raise value
        if callable(value):
            return value(**kwargs)
        defaults = {
            'runs': Handle([], {'total': 0}),
            'team_execution': Handle([], {'data': {'teamId': kwargs.get('team_id'), 'state': 'idle', 'queuePolicy': 'sequential', 'maxConcurrentRuns': 1, 'owners': [], **{key: [] for key in TEAM_CLASSES}}}),
            'team_tasks': Handle([], {'data': {'teamId': kwargs.get('team_id'), 'tasks': [], 'total': 0, 'limit': int(kwargs.get('query', {}).get('limit', '4')), 'offset': int(kwargs.get('query', {}).get('offset', '0'))}}),
            'teams': Handle([{'id': 'standing', 'lifetime': 'standing'}, {'id': 'delivery', 'lifetime': 'finite', 'enabled': True}]),
            'objectives': Handle([], {'data': {'rows': [{'id': 'T1', 'servedBy': [{'teamId': 'standing'}]}]}}),
            'efforts': Handle([], {'discovery': {'partial': True, 'lastSuccessfulScanAt': '2026-10-25T23:59:50Z'}, 'partial': True}),
            'goals': Handle([{'goal': {'name': 'goal', 'status': 'active'}}]),
            'backlog': Handle([], {'total': 0}),
            'usage': Handle([], {'programsDeclared': '0'}),
            'research': Handle([{'program': 'program-runtime.research-workspace-review', 'version': '1', 'status': 'ok', 'signals': {'unique_runs': 2}, 'errors': [], 'evidence': []}], {'digest': 'child-digest', 'version': '1'}),
        }
        return value if value is not None else defaults[name]
    def envelope(name, version):
        state.update(program=name, version=version, status='failed', phase='validate', inputs=inputs, signals={}, errors=[], evidence=[])
        return state
    def fail(status, klass, detail, where):
        state['status'] = status
        state['errors'].append({'class': klass, 'detail': detail, 'where': where})
        return 'report'
    def run(steps, step):
        while step:
            try:
                step = steps[step]()
            except Exception as exc:
                if step == 'report':
                    raise
                step = fail('failed', 'kernel_runtime', str(exc), step)
    program = SimpleNamespace(inputs=lambda: inputs, envelope=envelope, fail=fail, run=run,
                              child_envelope=lambda handle, **kw: handle.head(1)[0])
    namespace = {
        'program': program,
        'agent_manager': SimpleNamespace(run=SimpleNamespace(list=lambda **kw: read('runs', **kw)), effort=SimpleNamespace(board=lambda **kw: read('efforts', **kw))),
        'prompt_manager': SimpleNamespace(team=SimpleNamespace(list=lambda **kw: read('teams', **kw), execution_status=lambda **kw: read('team_execution', **kw), task_list=lambda **kw: read('team_tasks', **kw)), graph=SimpleNamespace(objectives=lambda **kw: read('objectives', **kw))),
        'swarm_manager': SimpleNamespace(goals=SimpleNamespace(list=lambda **kw: read('goals', **kw)), backlog=SimpleNamespace(list=lambda **kw: read('backlog', **kw))),
        'program_runtime': SimpleNamespace(programs=SimpleNamespace(portfolio=lambda **kw: read('usage', **kw))),
        'lib': SimpleNamespace(program_runtime=SimpleNamespace(research_workspace_review=lambda **kw: read('research', **kw))),
    }
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(SOURCE.read_text(), str(SOURCE), 'exec'), namespace)
    return json.loads(output.getvalue()), calls, output.getvalue()


def test_invalid_inputs_do_not_read_owners():
    for inputs in ({'row_limit': 0}, {'row_limit': True}, {'run_statuses': ['running', 'running']}, {'run_statuses': ['imagined']}, {'run_offsets': {'failed': 0}}, {'sample_offsets': {'teams': 200}}, {'research_root': 'relative'}, {'research_root': '/a\0b'}):
        result, calls, _ = execute(inputs)
        assert result['status'] == 'failed'
        assert result['errors'][0]['class'] == 'invalid_input'
        assert calls == []


def test_each_status_gets_one_explicit_page_and_no_liveness_is_inferred():
    result, calls, _ = execute({'run_statuses': ['running', 'unknown'], 'run_offsets': {'running': 8}},
                             {'runs': Handle([{'id': 'run', 'status': 'RUN_STATUS_RUNNING', 'executionMode': 'EXECUTION_MODE_IMPORTED'}], {'total': 99, 'hasMore': True})})
    requests = [kw for name, kw in calls if name == 'runs']
    assert requests == [{'status': 'RUN_STATUS_RUNNING', 'limit': 4, 'offset': 8}, {'status': 'RUN_STATUS_UNKNOWN', 'limit': 4, 'offset': 0}]
    source = result['signals']['sources']['runs/running']
    assert source['next_offset'] == 9 and source['query_complete'] is False
    assert source['entities'][0]['execution_liveness'] == 'unverified'
    assert source['entities'][0]['accepted_outcome'] == 'unverified'


def test_standing_and_finite_teams_are_separate_from_executor_counts():
    result, _, _ = execute()
    teams = result['signals']['sources']['teams']
    assert teams['lifetime_counts'] == {'standing': 1, 'finite': 1}
    assert teams['entities'][0]['enabled'] is False
    assert all(r['execution_liveness'] == 'not_checked' for r in teams['entities'])
    assert result['signals']['reconciliation']['relations'] == []
    assert result['signals']['sources']['swarm_executions']['status'] == 'not_collected'
    assert result['status'] == 'partial'


def test_unknown_source_is_not_empty_success_and_siblings_survive():
    result, _, _ = execute(overrides={'teams': RuntimeError('owner unavailable')})
    source = result['signals']['sources']['teams']
    assert source['status'] == 'unknown' and source['response_rows'] is None
    assert source['query_complete'] is False
    assert result['signals']['sources']['swarm_goals']['response_rows'] == 1


def test_explicit_null_counts_are_rejected_but_proto_omitted_zero_is_valid():
    for bad in (None, True, 0.5, 'bad'):
        result, _, _ = execute({'run_statuses': ['running']}, {'runs': Handle([], {'total': bad})})
        source = result['signals']['sources']['runs/running']
        assert source['status'] == 'unknown'
        assert source['errors'][0]['class'] == 'invalid_response'
    result, _, _ = execute({'run_statuses': ['running']}, {'runs': Handle([], {})})
    assert result['signals']['sources']['runs/running']['total_in_scope'] == 0


def test_owner_pagination_cannot_claim_complete_below_total():
    result, _, _ = execute({'run_statuses': ['running']}, {'runs': Handle([], {'total': 10, 'hasMore': False})})
    assert result['signals']['sources']['runs/running']['status'] == 'unknown'


def test_null_team_boolean_is_not_disabled():
    result, _, _ = execute(overrides={'teams': Handle([{'id': 'team', 'enabled': None}])})
    assert result['signals']['sources']['teams']['status'] == 'unknown'


def test_effort_discovery_age_survives_fresh_collection():
    result, _, _ = execute()
    source = result['signals']['sources']['efforts']
    assert source['discovery']['lastSuccessfulScanAt'] == '2026-10-25T23:59:50Z'
    assert source['query_complete'] is False
    assert source['producer_observed_at'] is None


def test_sample_offsets_disclose_output_coverage():
    result, _, _ = execute({'row_limit': 1, 'sample_offsets': {'teams': 1}})
    source = result['signals']['sources']['teams']
    assert source['query_complete'] is True and source['output_complete'] is False
    assert source['entities'][0]['id'] == 'delivery'


def test_missing_research_is_unavailable_and_does_not_create_zero_inventory(tmp_path):
    child = Handle([{'status': 'unavailable', 'signals': {'unique_runs': 0}, 'errors': [{'class': 'workspace_unavailable'}]}])
    result, _, _ = execute({'research_root': str(tmp_path)}, {'research': child})
    sources = result['signals']['sources']
    assert sources['research_catalog']['status'] == 'unavailable'
    assert 'summary' not in sources['research_catalog']
    assert sources['research_workspace']['status'] == 'unavailable'
    assert sources['research_workspace']['query_complete'] is False


def test_research_catalog_and_directory_counts_keep_distinct_denominators(tmp_path):
    (tmp_path / 'CATALOG.json').write_text(json.dumps({'version': 2, 'updated_at': '2026-11-05T00:00:00Z', 'topics': [{'id': 'topic'}], 'runs': [{'id': 'study', 'kind': 'study', 'workspace': 'where'}]}))
    result, _, _ = execute({'research_root': str(tmp_path)})
    sources = result['signals']['sources']
    assert sources['research_catalog']['summary']['runs_retained_entries'] == 1
    assert sources['research_workspace']['signals']['unique_runs'] == 2
    assert sources['research_catalog']['producer_observed_at'] == '2026-11-05T00:00:00Z'
    assert sources['research_workspace']['artifact']['digest'] == 'child-digest'


def test_catalog_symlink_outside_authorized_root_is_not_read(tmp_path):
    root = tmp_path / 'root'
    root.mkdir()
    outside = tmp_path / 'outside.json'
    outside.write_text('{"topics": [], "runs": []}')
    (root / 'CATALOG.json').symlink_to(outside)
    result, _, _ = execute({'research_root': str(root)})
    assert result['signals']['sources']['research_catalog']['status'] == 'unknown'
    assert 'content_sha256' not in result['signals']['sources']['research_catalog']


def test_large_source_is_bounded_with_explicit_omissions():
    big = {str(n): ['large' * 1000] * 100 for n in range(100)}
    result, _, stdout = execute({'row_limit': 20}, {'objectives': Handle([], {'data': {'rows': [], 'validation': big, 'drift': big}})})
    assert len(stdout.encode()) < 65536
    assert result['signals']['output_truncated'] is True
    assert result['signals']['sources']['objectives']['details_omitted_due_to_output_limit']


def test_samples_stop_at_inspection_bound_without_an_invalid_cursor():
    result, _, _ = execute({'sample_offsets': {'teams': 199}}, {'teams': Handle([{'id': str(n)} for n in range(210)])})
    source = result['signals']['sources']['teams']
    assert source['returned_rows'] == 1
    assert source['next_sample_offset'] is None
    assert source['inspection_bound_reached'] is True
    assert source['summary_complete'] is False


def test_active_goal_examples_do_not_hide_archived_denominator():
    goals = Handle([{'goal': {'name': 'old', 'status': 'archived', 'updated': '2026-11-06'}}, {'goal': {'name': 'active', 'status': 'active', 'updated': '2026-10-01'}}])
    result, _, _ = execute({'row_limit': 1}, {'goals': goals})
    source = result['signals']['sources']['swarm_goals']
    assert source['entities'][0]['name'] == 'active'
    assert source['status_counts'] == {'archived': 1, 'active': 1}


def test_unselected_teams_do_not_trigger_queue_or_board_reads():
    result, calls, _ = execute()
    assert not any(name in ('team_execution', 'team_tasks') for name, _ in calls)
    assert result['signals']['sources']['team_activity']['status'] == 'not_requested'
    assert result['signals']['coverage']['selected_team_ids'] == []


def test_invalid_team_selection_rejects_before_any_owner_read():
    for selected in ('team', [None], [''], ['x', 'x'], ['../x'], ['a/b'], ['a b'], ['a', 'b', 'c', 'd', 'e'], ['x' * 129]):
        result, calls, _ = execute({'team_ids': selected})
        assert result['status'] == 'failed' and not calls
    for offsets in ({'other': 0}, {'one': -1}, {'one': True}, {'one': 1000001}, None):
        result, calls, _ = execute({'team_ids': ['one'], 'team_task_offsets': offsets})
        assert result['status'] == 'failed' and not calls


def execution_data(team='finite', **extra):
    data = {'teamId': team, 'state': 'active', 'queuePolicy': 'bounded-parallel', 'maxConcurrentRuns': 3,
            'owners': [{'agentId': 'member', 'phase': 'retained', 'state': 'approval_pending', 'taskId': 'task', 'runId': 'run', 'ownerStartedAt': '2026-10-01T00:00:00Z'}],
            **{key: [] for key in TEAM_CLASSES}}
    data.update(runningAgentIds=['member'], approvalPendingAgentIds=['member'])
    data.update(extra)
    return Handle([], {'data': data})


def test_selected_team_ownership_is_not_native_liveness_or_acceptance():
    result, calls, _ = execute({'team_ids': ['finite']}, {'team_execution': execution_data()})
    assert [kw for name, kw in calls if name == 'team_execution'] == [{'team_id': 'finite'}]
    assert [kw for name, kw in calls if name == 'team_tasks'] == [{'team_id': 'finite', 'query': {'limit': '4', 'offset': '0'}}]
    item = result['signals']['sources']['team_execution/finite']
    assert item['status'] == 'ok' and item['query_complete']
    assert item['owner_class_counts']['runningAgentIds'] == 1
    assert item['owner_class_counts']['approvalPendingAgentIds'] == 1
    assert item['entities'][0]['runId'] == 'run'
    assert item['entities'][0]['ownerStartedAt'] == '2026-10-01T00:00:00Z'
    assert item['entities'][0]['execution_liveness'] == 'unverified'
    assert item['entities'][0]['accepted_outcome'] == 'unverified'
    assert item['producer_observed_at'] is None
    assert result['status'] == 'partial' and result['signals']['reconciliation']['relations'] == []


def test_selected_team_reads_fail_independently():
    result, _, _ = execute({'team_ids': ['finite', 'standing']},
                          {'team_execution': lambda **kw: execution_data(team=kw['team_id']) if kw['team_id'] == 'standing' else (_ for _ in ()).throw(RuntimeError('unavailable'))})
    sources = result['signals']['sources']
    assert sources['team_execution/finite']['status'] == 'unknown'
    assert sources['team_tasks/finite']['status'] == 'ok'
    assert sources['team_execution/standing']['status'] == 'ok'


def test_missing_or_null_struct_fields_never_become_zero():
    for field in ('queue', 'runningAgentIds', 'owners', 'maxConcurrentRuns'):
        for absent in (True, False):
            value = execution_data()
            if absent:
                del value.metadata['data'][field]
            else:
                value.metadata['data'][field] = None
            result, _, _ = execute({'team_ids': ['finite']}, {'team_execution': value})
            assert result['signals']['sources']['team_execution/finite']['status'] == 'unknown'
    for field in ('total', 'limit', 'offset', 'tasks'):
        value = Handle([], {'data': {'teamId': 'finite', 'tasks': [], 'total': 0, 'limit': 4, 'offset': 0}})
        del value.metadata['data'][field]
        result, _, _ = execute({'team_ids': ['finite']}, {'team_tasks': value})
        assert result['signals']['sources']['team_tasks/finite']['status'] == 'unknown'


def test_owner_team_mismatch_and_task_pagination_are_rejected():
    for overrides in ({'team_execution': execution_data(team='other')},
                      {'team_tasks': Handle([], {'data': {'teamId': 'finite', 'tasks': [], 'total': 2, 'limit': 4, 'offset': 0}})},
                      {'team_tasks': Handle([], {'data': {'teamId': 'other', 'tasks': [], 'total': 0, 'limit': 4, 'offset': 0}})}):
        result, _, _ = execute({'team_ids': ['finite']}, overrides)
        key = 'team_execution/finite' if 'team_execution' in overrides else 'team_tasks/finite'
        assert result['signals']['sources'][key]['status'] == 'unknown'


def test_task_page_retains_exact_identity_staleness_and_cursor():
    long_id = 'task-' + 'x' * 300
    value = Handle([], {'data': {'teamId': 'finite', 'tasks': [{'id': long_id, 'status': 'done', 'updatedAt': '2026-09-01'}], 'total': 10, 'limit': 1, 'offset': 3}})
    result, _, _ = execute({'team_ids': ['finite'], 'row_limit': 1, 'team_task_offsets': {'finite': 3}}, {'team_tasks': value})
    item = result['signals']['sources']['team_tasks/finite']
    assert item['next_offset'] == 4 and item['query_complete'] is False
    assert item['entities'][0]['id'] == long_id
    assert item['entities'][0]['updatedAt'] == '2026-09-01'
    assert item['entities'][0]['accepted_outcome'] == 'unverified'


def test_team_queue_samples_are_bounded_without_hiding_denominators():
    value = execution_data(queue=['one', 'two', 'three'], owners=[{'agentId': x, 'phase': 'queued'} for x in ('one', 'two', 'three')])
    result, _, _ = execute({'team_ids': ['finite'], 'row_limit': 1}, {'team_execution': value})
    item = result['signals']['sources']['team_execution/finite']
    assert item['response_rows'] == 3 and item['owners_omitted'] == 2
    assert item['queue_sample'] == ['one'] and item['queue_members_omitted'] == 2
    assert item['owner_class_counts']['queue'] == 3 and item['output_complete'] is False


def test_team_execution_manifest_exposes_only_existing_read_rpc():
    manifest = json.loads((SOURCE.parents[3] / 'prompt-manager/cli/manifest.json').read_text())
    team = next(g for g in manifest['groups'] if g['name'] == 'team')
    command = next(c for c in team['commands'] if c['name'] == 'execution-status')
    assert command['binding'] == {'kind': 'connect-rpc', 'service': 'HeartbeatService', 'method': 'GetTeamExecutionStatus'}
    assert command['governance'] == {'effect': 'read', 'run_eligible': True, 'permissions': ['network:internal']}
    assert not any(x.get('method') == 'GetTeamExecutionStatus' for x in manifest.get('omitted', []))


def test_large_team_sources_fit_tier_and_exclude_private_payloads():
    identity = 'x' * 1000
    def big(**kw):
        return execution_data(team=kw['team_id'], queue=[identity] * 20,
                              owners=[{'agentId': identity, 'phase': 'retained', 'runId': identity, 'taskId': identity, 'frozenPrompt': 'PRIVATE-PAYLOAD'}] * 20)
    result, _, stdout = execute({'team_ids': ['one', 'two', 'three', 'four'], 'row_limit': 20}, {'team_execution': big})
    assert len(stdout.encode()) < 65536
    assert 'PRIVATE-PAYLOAD' not in stdout
    assert result['signals']['output_truncated']
    assert any('omitted_due_to_output_limit' in x for x in result['signals']['sources'].values())


def test_queue_sample_omission_is_incomplete_even_when_owner_list_is_empty():
    result, _, _ = execute({'team_ids': ['finite'], 'row_limit': 1}, {'team_execution': execution_data(owners=[], queue=['one', 'two'])})
    assert result['signals']['sources']['team_execution/finite']['output_complete'] is False
