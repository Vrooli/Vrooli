"""Attention index fails closed on ambiguous or unavailable snapshots."""
from types import SimpleNamespace

from program_fixture import Child, run


def execute(root):
    return run('research-attention-index', {'research_root': str(root)})


def parent(inputs, **children):
    """research-supervision-read with each named child returning its fixture envelope."""
    library = SimpleNamespace(program_runtime=SimpleNamespace(**{
        name.replace('-', '_'): (lambda envelope, name=name: lambda **kwargs: Child(name, envelope))(envelope)
        for name, envelope in children.items()}))
    return run('research-supervision-read', inputs, lib=library)


def test_empty_and_actionable_snapshot(tmp_path):
    path = tmp_path / 'ATTENTION.md'
    path.write_text('# Index\n## Actionable now\n\nNo actionable items are recorded.\n\n## Future gates\n### OLD — deferred\n')
    result = execute(tmp_path)
    assert result['signals']['actionable_ids'] == []
    assert result['status'] == 'ok'
    path.write_text('## Actionable now\n### WALLET-123 — Choose a route\n- private context\n\n### DEBT_2 — Review\n- more context\n## Deferred\n')
    result = execute(tmp_path)
    assert result['signals']['actionable_ids'] == ['WALLET-123', 'DEBT_2']
    assert 'private context' not in str(result)


def test_unknown_for_missing_ambiguous_duplicate_and_symlink(tmp_path):
    path = tmp_path / 'ATTENTION.md'
    assert execute(tmp_path)['signals']['coverage'] == 'unknown'
    for text in ['## Actionable now\nNothing here\n', '## Actionable now\n### BAD HEADING\n', '## Actionable now\n### X — one\n### X — duplicate\n', '## Actionable now\nNo actionable items are recorded.\n## Actionable now\nNo actionable items are recorded.\n']:
        path.write_text(text)
        assert execute(tmp_path)['signals']['coverage'] == 'unknown'
    path.unlink()
    other = tmp_path / 'other.md'
    other.write_text('## Actionable now\nNo actionable items are recorded.\n')
    path.symlink_to(other)
    assert execute(tmp_path)['signals']['coverage'] == 'unknown'


def test_parent_flags_delivery_not_checked_without_selected_effort():
    index = {'status': 'ok', 'signals': {'actionable_ids': ['WALLET-123']}, 'errors': [], 'evidence': []}
    result = parent({'research_root': '/authorized/research'}, **{'research-attention-index': index})
    assert {'source': 'attention_index', 'subject': 'WALLET-123', 'signal': 'delivery_not_checked'} in result['signals']['attention']


def test_parent_flags_unobserved_decision_when_effort_selected():
    index = {'status': 'ok', 'signals': {'actionable_ids': ['WALLET-123']}, 'errors': [], 'evidence': []}
    delivery = {'status': 'partial', 'signals': {'items': []}, 'errors': [], 'evidence': []}
    result = parent({'research_root': '/authorized/research', 'effort_ref': 'effort:test'},
                    **{'research-attention-index': index, 'research-attention-delivery': delivery})
    assert {'source': 'attention_index', 'subject': 'WALLET-123', 'signal': 'decision_unlinked_or_unobserved'} in result['signals']['attention']


def test_parent_reports_unavailable_index():
    index = {'status': 'unavailable', 'signals': {'coverage': 'unknown', 'actionable_ids': []}, 'errors': [], 'evidence': []}
    result = parent({'research_root': '/unavailable/research'}, **{'research-attention-index': index})
    assert {'source': 'attention_index', 'subject': '', 'signal': 'index_unavailable'} in result['signals']['attention']


PAGE = ('# Research\n\nHeader line.\n\n## Proposals awaiting approval\n\n{proposals}\n\n'
        '## Actionable now\n\n{actionable}\n\n## Agent prerequisites and future gates\n\n| ID | State |\n|---|---|\n')


def test_proposals_are_reported_beside_actionable_items(tmp_path):
    path = tmp_path / 'ATTENTION.md'
    path.write_text(PAGE.format(proposals='No proposals are recorded.', actionable='No actionable items are recorded.'))
    result = execute(tmp_path)
    assert result['status'] == 'ok'
    assert result['signals']['proposal_coverage'] == 'parsed_snapshot'
    assert result['signals']['proposal_ids'] == []
    assert result['signals']['actionable_ids'] == []
    path.write_text(PAGE.format(
        proposals='### PROPOSAL-MUSIC-STUDIO — Build a studio\n- What gets built: private detail\n\n### PROPOSAL-DEBT-small-audit — Smaller\n- detail',
        actionable='### WALLET-1 — Choose a route\n- context'))
    result = execute(tmp_path)
    assert result['signals']['proposal_ids'] == ['PROPOSAL-MUSIC-STUDIO', 'PROPOSAL-DEBT-small-audit']
    assert result['signals']['actionable_ids'] == ['WALLET-1']
    assert 'private detail' not in str(result)


def test_unparsed_proposals_are_unknown_without_losing_actionable_snapshot(tmp_path):
    path = tmp_path / 'ATTENTION.md'
    cases = [
        '## Actionable now\n### WALLET-1 — Choose\n',  # older index without the section
        PAGE.format(proposals='', actionable='### WALLET-1 — Choose'),  # empty without marker
        PAGE.format(proposals='### MUSIC-STUDIO — Missing prefix', actionable='### WALLET-1 — Choose'),
        PAGE.format(proposals='Unclassified prose\n### PROPOSAL-A — Build', actionable='### WALLET-1 — Choose'),
        PAGE.format(proposals='### PROPOSAL-A — Build\n### PROPOSAL-A — Again', actionable='### WALLET-1 — Choose'),
        PAGE.format(proposals='No proposals are recorded.', actionable='### WALLET-1 — Choose') + '## Proposals awaiting approval\nNo proposals are recorded.\n',
    ]
    for text in cases:
        path.write_text(text)
        result = execute(tmp_path)
        assert result['status'] == 'ok', text
        assert result['signals']['actionable_ids'] == ['WALLET-1'], text
        assert result['signals']['proposal_coverage'] == 'unknown', text
        assert result['signals']['proposal_ids'] == [], text
