"""Behavior checks for the bounded decision-to-delivery reconciliation."""
import contextlib
import io
import json
import runpy
from pathlib import Path
from types import SimpleNamespace

SOURCE = Path(__file__).resolve().parents[2] / '.vrooli/program-runtime/research-attention-delivery.py'

class Handle:
    def __init__(self, value): self.value = value
    def raw(self): return self.value

class Runtime:
    def __init__(self): self.result = None
    def inputs(self): return getattr(self, 'data', {'effort_ref': 'effort:test'})
    def envelope(self, name, version, **kwargs):
        self.result = {'program': name, 'version': version, 'status': 'failed', 'phase': 'validate', 'inputs': {}, 'signals': {}, 'errors': [], 'evidence': []}
        return self.result
    def fail(self, status, kind, detail, where):
        self.result.update(status=status, phase=where)
        self.result['errors'].append({'class': kind, 'detail': detail, 'where': where})
        return 'report'
    def run(self, states, start):
        while start: start = states[start]()


def execute(decision, asks, notice=None, *, partial=False, needs=None, next_page=''):
    runtime = Runtime()
    runtime.data = {'effort_ref': 'effort:test', 'include_board': needs is not None}
    board = {'rows': [{'enrollment': {'effortRef': 'effort:test'}, 'goalHome': {'state': 'available', 'openDecisions': [decision], 'needsOperator': needs or []}}], 'partial': partial}
    owner_decision = dict(decision, effortRef='effort:test')
    def decisions_page(**kwargs):
        assert kwargs == {'effort_ref': 'effort:test', 'limit': 16, 'page_token': ''}
        return Handle({'decisions': [owner_decision], 'nextPageToken': next_page})
    agent = SimpleNamespace(effort=SimpleNamespace(board=lambda **kw: Handle(board), decisions_page=decisions_page))
    hub = SimpleNamespace(conversations=SimpleNamespace(list=lambda **kw: Handle(asks)), notifications=SimpleNamespace(get=lambda **kw: Handle(notice)))
    with contextlib.redirect_stdout(io.StringIO()):
        runpy.run_path(str(SOURCE), init_globals={'program': runtime, 'agent_manager': agent, 'notification_hub': hub})
    return runtime.result


def test_delivered_requires_matching_receipt():
    decision = {'decisionId': 'd', 'state': 'pending', 'eventId': 'event', 'publishedAt': '2026-01-01T00:00:00Z'}
    ask = {'id': 'ask', 'requesterRef': 'event', 'source': 'agent-manager', 'sourceEventType': 'agent_manager.effort.decision_requested.v1', 'notificationId': 'n', 'state': 'pending'}
    result = execute(decision, {'asks': [ask]}, {'notification': {'id': 'n', 'state': 'NOTIFICATION_STATE_DELIVERED'}, 'receipts': [{'notificationId': 'n', 'deliveredAt': '2026-01-01T00:00:01Z'}]})
    assert result['signals']['items'][0]['delivered_receipt'] is True
    assert result['status'] == 'partial'  # narrative attention index remains uncovered
    result = execute(decision, {'asks': [ask]}, {'notification': {'id': 'n', 'state': 'NOTIFICATION_STATE_DELIVERED'}, 'receipts': []})
    assert result['signals']['items'][0]['delivered_receipt'] is False


def test_answered_ask_with_pending_owner_decision_requires_reconciliation():
    decision = {'decisionId': 'd', 'state': 'pending', 'eventId': 'event', 'publishedAt': 'now'}
    ask = {'id': 'ask', 'requesterRef': 'event', 'source': 'agent-manager', 'sourceEventType': 'agent_manager.effort.decision_requested.v1', 'notificationId': 'n', 'state': 'answered'}
    notice = {'notification': {'id': 'n', 'state': 'NOTIFICATION_STATE_DELIVERED'}, 'receipts': [{'notificationId': 'n', 'deliveredAt': 'now'}]}
    item = execute(decision, {'asks': [ask]}, notice)['signals']['items'][0]
    assert item['delivered_receipt'] is True
    assert item['resolution'] == 'ask_terminal_decision_pending'


def test_unpublished_and_incomplete_search_remain_distinct():
    base = {'decisionId': 'd', 'state': 'pending', 'eventId': 'event'}
    assert execute(base, {'asks': []})['signals']['items'][0]['delivery'] == 'unpublished'
    base['publishedAt'] = '2026-01-01T00:00:00Z'
    assert execute(base, {'asks': [], 'nextCursor': 'more'})['signals']['items'][0]['delivery'] == 'unknown_incomplete_search'
    assert execute(base, {'asks': []})['signals']['items'][0]['delivery'] == 'ask_not_observed'
    paged = execute(base, {'asks': []}, next_page='next')
    assert paged['signals']['next_page_token'] == 'next'
    assert paged['signals']['coverage'] == 'partial'


def test_other_source_and_mismatched_notification_do_not_prove_delivery():
    decision = {'decisionId': 'd', 'state': 'pending', 'eventId': 'event', 'publishedAt': 'now'}
    unrelated = {'id': 'ask', 'requesterRef': 'event', 'source': 'other', 'sourceEventType': 'agent_manager.effort.decision_requested.v1', 'notificationId': 'n'}
    assert execute(decision, {'asks': [unrelated]})['signals']['items'][0]['delivery'] == 'ask_not_observed'
    unrelated['source'] = 'agent-manager'
    assert execute(decision, {'asks': [unrelated]}, {'notification': {'id': 'wrong'}})['signals']['items'][0]['delivery'] == 'unknown_owner_read'


def test_missing_decision_state_is_unknown_not_closed():
    result = execute({'decisionId': 'd', 'eventId': 'event'}, {'asks': []})
    assert result['signals']['items'][0]['delivery'] == 'unknown_decision_state'
    assert result['signals']['coverage'] == 'partial'


def test_unlinked_attention_is_exposed_without_private_content():
    decision = {'decisionId': 'd', 'state': 'pending', 'eventId': 'event'}
    result = execute(decision, {'asks': []}, needs=['- [decision d] known', '- sensitive unlinked question'])
    item = result['signals']['items'][-1]
    assert item['delivery'] == 'unknown_unlinked_attention'
    assert item['count'] == 1
    assert 'sensitive' not in json.dumps(result)


def test_supervision_parent_surfaces_unpublished_decision():
    parent = SOURCE.with_name('research-supervision-read.py')
    child = {'status': 'partial', 'signals': {'items': [{'decision_id': 'd', 'delivery': 'unpublished'}], 'attention_view_coverage': 'unknown'}, 'errors': [], 'evidence': []}
    class ChildHandle:
        def meta(self): return {'program': 'program-runtime.research-attention-delivery', 'version': '1', 'digest': 'fixture'}
    class ParentRuntime(Runtime):
        def child_envelope(self, handle, **kwargs): return child
    runtime = ParentRuntime()
    library = SimpleNamespace(program_runtime=SimpleNamespace(research_attention_delivery=lambda **kw: ChildHandle()))
    with contextlib.redirect_stdout(io.StringIO()):
        runpy.run_path(str(parent), init_globals={'program': runtime, 'lib': library})
    assert runtime.result['status'] == 'partial'
    assert {'source': 'attention_delivery', 'subject': 'd', 'signal': 'unpublished'} in runtime.result['signals']['attention']
    assert {'source': 'attention_index', 'subject': '', 'signal': 'index_not_requested'} in runtime.result['signals']['attention']
    assert runtime.result['signals']['sources']['attention_delivery']['signals']['attention_view_coverage'] == 'unknown'


def test_supervision_parent_surfaces_unavailable_delivery_review():
    parent = SOURCE.with_name('research-supervision-read.py')
    child = {'status': 'unavailable', 'signals': {'items': []}, 'errors': [], 'evidence': []}
    class ChildHandle:
        def meta(self): return {}
    class ParentRuntime(Runtime):
        def child_envelope(self, handle, **kwargs): return child
    runtime = ParentRuntime()
    library = SimpleNamespace(program_runtime=SimpleNamespace(research_attention_delivery=lambda **kw: ChildHandle()))
    with contextlib.redirect_stdout(io.StringIO()):
        runpy.run_path(str(parent), init_globals={'program': runtime, 'lib': library})
    assert runtime.result['signals']['attention'][0]['signal'] == 'delivery_review_unavailable'
    assert runtime.result['status'] == 'partial'


def test_supervision_parent_surfaces_answer_even_when_delivered():
    parent = SOURCE.with_name('research-supervision-read.py')
    child = {'status': 'partial', 'signals': {'items': [{'decision_id': 'd', 'delivery': 'NOTIFICATION_STATE_DELIVERED', 'delivered_receipt': True, 'resolution': 'ask_terminal_decision_pending'}]}, 'errors': [], 'evidence': []}
    class ChildHandle:
        def meta(self): return {}
    class ParentRuntime(Runtime):
        def child_envelope(self, handle, **kwargs): return child
    runtime = ParentRuntime()
    library = SimpleNamespace(program_runtime=SimpleNamespace(research_attention_delivery=lambda **kw: ChildHandle()))
    with contextlib.redirect_stdout(io.StringIO()):
        runpy.run_path(str(parent), init_globals={'program': runtime, 'lib': library})
    assert {'source': 'attention_delivery', 'subject': 'd', 'signal': 'ask_terminal_decision_pending'} in runtime.result['signals']['attention']


def test_supervision_parent_forwards_owner_cursor_and_board_choice():
    parent = SOURCE.with_name('research-supervision-read.py')
    calls = []
    child = {'status': 'partial', 'signals': {'items': []}, 'errors': [], 'evidence': []}
    class ChildHandle:
        def meta(self): return {}
    class ParentRuntime(Runtime):
        def child_envelope(self, handle, **kwargs): return child
    runtime = ParentRuntime()
    runtime.data = {'effort_ref': 'effort:test', 'decision_page_token': 'opaque', 'include_effort_board': True}
    def invoke(**kwargs):
        calls.append(kwargs)
        return ChildHandle()
    library = SimpleNamespace(program_runtime=SimpleNamespace(research_attention_delivery=invoke))
    with contextlib.redirect_stdout(io.StringIO()):
        runpy.run_path(str(parent), init_globals={'program': runtime, 'lib': library})
    assert calls == [{'effort_ref': 'effort:test', 'decision_page_token': 'opaque', 'include_board': True}]
