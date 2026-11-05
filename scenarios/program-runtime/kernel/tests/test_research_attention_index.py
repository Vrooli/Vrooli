"""Attention index fails closed on ambiguous or unavailable snapshots."""
import contextlib
import io
import runpy
from pathlib import Path
from types import SimpleNamespace

SOURCE = Path(__file__).resolve().parents[2] / '.vrooli/program-runtime/research-attention-index.py'
PARENT = SOURCE.with_name('research-supervision-read.py')

class Runtime:
    def __init__(self, data): self.data = data
    def inputs(self): return self.data
    def envelope(self, name, version, **kwargs):
        self.result = {'program': name, 'version': version, 'status': 'failed', 'phase': 'validate', 'inputs': {}, 'signals': {}, 'errors': [], 'evidence': []}
        return self.result
    def fail(self, status, kind, detail, where):
        self.result.update(status=status, phase=where)
        self.result['errors'].append({'class': kind, 'detail': detail})
        return 'report'
    def run(self, states, start):
        while start: start = states[start]()


def execute(root):
    runtime = Runtime({'research_root': str(root)})
    with contextlib.redirect_stdout(io.StringIO()):
        runpy.run_path(str(SOURCE), init_globals={'program': runtime})
    return runtime.result


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
    class Handle:
        def meta(self): return {}
    class ParentRuntime(Runtime):
        def child_envelope(self, handle, **kwargs): return index
    runtime = ParentRuntime({'research_root': '/authorized/research'})
    library = SimpleNamespace(program_runtime=SimpleNamespace(research_attention_index=lambda **kwargs: Handle()))
    with contextlib.redirect_stdout(io.StringIO()):
        runpy.run_path(str(PARENT), init_globals={'program': runtime, 'lib': library})
    assert {'source': 'attention_index', 'subject': 'WALLET-123', 'signal': 'delivery_not_checked'} in runtime.result['signals']['attention']


def test_parent_flags_unobserved_decision_when_effort_selected():
    index = {'status': 'ok', 'signals': {'actionable_ids': ['WALLET-123']}, 'errors': [], 'evidence': []}
    delivery = {'status': 'partial', 'signals': {'items': []}, 'errors': [], 'evidence': []}
    class Handle:
        def __init__(self, value): self.value = value
        def meta(self): return {}
    class ParentRuntime(Runtime):
        def child_envelope(self, handle, **kwargs): return handle.value
    runtime = ParentRuntime({'research_root': '/authorized/research', 'effort_ref': 'effort:test'})
    library = SimpleNamespace(program_runtime=SimpleNamespace(
        research_attention_index=lambda **kwargs: Handle(index),
        research_attention_delivery=lambda **kwargs: Handle(delivery)))
    with contextlib.redirect_stdout(io.StringIO()):
        runpy.run_path(str(PARENT), init_globals={'program': runtime, 'lib': library})
    assert {'source': 'attention_index', 'subject': 'WALLET-123', 'signal': 'decision_unlinked_or_unobserved'} in runtime.result['signals']['attention']


def test_parent_reports_unavailable_index():
    index = {'status': 'unavailable', 'signals': {'coverage': 'unknown', 'actionable_ids': []}, 'errors': [], 'evidence': []}
    class Handle:
        def meta(self): return {}
    class ParentRuntime(Runtime):
        def child_envelope(self, handle, **kwargs): return index
    runtime = ParentRuntime({'research_root': '/unavailable/research'})
    library = SimpleNamespace(program_runtime=SimpleNamespace(research_attention_index=lambda **kwargs: Handle()))
    with contextlib.redirect_stdout(io.StringIO()):
        runpy.run_path(str(PARENT), init_globals={'program': runtime, 'lib': library})
    assert {'source': 'attention_index', 'subject': '', 'signal': 'index_unavailable'} in runtime.result['signals']['attention']
