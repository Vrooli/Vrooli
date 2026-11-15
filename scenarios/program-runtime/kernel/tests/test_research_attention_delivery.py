"""Behavior checks for the bounded decision-to-delivery reconciliation."""
import json
from types import SimpleNamespace

from program_fixture import Child, run


class Handle:
    def __init__(self, value): self.value = value
    def raw(self): return self.value


def execute(decision, asks, notice=None, *, partial=False, needs=None, next_page=''):
    board = {'rows': [{'enrollment': {'effortRef': 'effort:test'}, 'goalHome': {'state': 'available', 'openDecisions': [decision], 'needsOperator': needs or []}}], 'partial': partial}
    owner_decision = dict(decision, effortRef='effort:test')
    def decisions_page(**kwargs):
        assert kwargs == {'effort_ref': 'effort:test', 'limit': 16, 'page_token': ''}
        return Handle({'decisions': [owner_decision], 'nextPageToken': next_page})
    agent = SimpleNamespace(effort=SimpleNamespace(board=lambda **kw: Handle(board), decisions_page=decisions_page))
    hub = SimpleNamespace(conversations=SimpleNamespace(list=lambda **kw: Handle(asks)), notifications=SimpleNamespace(get=lambda **kw: Handle(notice)))
    return run('research-attention-delivery', {'effort_ref': 'effort:test', 'include_board': needs is not None}, agent_manager=agent, notification_hub=hub)


def supervise(child, inputs=None, calls=None):
    """research-supervision-read over one attention-delivery child envelope."""
    def invoke(**kwargs):
        if calls is not None:
            calls.append(kwargs)
        return Child('research-attention-delivery', child)
    library = SimpleNamespace(program_runtime=SimpleNamespace(research_attention_delivery=invoke))
    return run('research-supervision-read', inputs or {'effort_ref': 'effort:test'}, lib=library)


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
    result = supervise({'status': 'partial', 'signals': {'items': [{'decision_id': 'd', 'delivery': 'unpublished'}], 'attention_view_coverage': 'unknown'}, 'errors': [], 'evidence': []})
    assert result['status'] == 'partial'
    assert {'source': 'attention_delivery', 'subject': 'd', 'signal': 'unpublished'} in result['signals']['attention']
    assert {'source': 'attention_index', 'subject': '', 'signal': 'index_not_requested'} in result['signals']['attention']
    assert result['signals']['sources']['attention_delivery']['signals']['attention_view_coverage'] == 'unknown'


def test_supervision_parent_surfaces_unavailable_delivery_review():
    result = supervise({'status': 'unavailable', 'signals': {'items': []}, 'errors': [], 'evidence': []})
    assert result['signals']['attention'][0]['signal'] == 'delivery_review_unavailable'
    assert result['status'] == 'partial'


def test_supervision_parent_surfaces_answer_even_when_delivered():
    result = supervise({'status': 'partial', 'signals': {'items': [{'decision_id': 'd', 'delivery': 'NOTIFICATION_STATE_DELIVERED', 'delivered_receipt': True, 'resolution': 'ask_terminal_decision_pending'}]}, 'errors': [], 'evidence': []})
    assert {'source': 'attention_delivery', 'subject': 'd', 'signal': 'ask_terminal_decision_pending'} in result['signals']['attention']


def test_supervision_parent_forwards_owner_cursor_and_board_choice():
    calls = []
    supervise({'status': 'partial', 'signals': {'items': []}, 'errors': [], 'evidence': []},
              {'effort_ref': 'effort:test', 'decision_page_token': 'opaque', 'include_effort_board': True}, calls)
    assert calls == [{'effort_ref': 'effort:test', 'decision_page_token': 'opaque', 'include_board': True}]
