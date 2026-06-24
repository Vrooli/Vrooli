import { describe, expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { DecisionTraceList } from './DecisionTracePanel';
import type { DecisionTraceEntry } from '@/types/api';

describe('DecisionTraceList rendering', () => {
  it('renders the core per-iteration signals: skill, dimension, score, delta, halt, rationale', () => {
    const entries: DecisionTraceEntry[] = [
      {
        iteration: 1,
        chosen_skill: 'refactor',
        heaviest_dimension: 'standards',
        dimension_scores: { standards: 8 },
        score_before: 8,
        score_after: 12,
        realized_delta: -4,
        halt_reason: 'thrashing_cycle',
        rationale: 'selected "refactor" for the heaviest open dimension',
      },
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={entries} />);

    expect(html).toContain('refactor');
    expect(html).toContain('standards');
    expect(html).toContain('Δ +4.0 (regressed)');
    expect(html).toContain('halt: thrashing_cycle');
    expect(html).toContain('heaviest open dimension');
  });

  it('renders an improvement as a negative weighted-score delta', () => {
    const entries: DecisionTraceEntry[] = [
      {
        iteration: 1,
        chosen_skill: 'lint-fix',
        heaviest_dimension: 'standards',
        score_before: 12,
        score_after: 8,
        realized_delta: 4,
      },
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={entries} />);
    expect(html).toContain('Δ −4.0 (improved)');
  });

  it('flags a gamed iteration as a promote-safety warning', () => {
    const gamed: DecisionTraceEntry[] = [
      {
        iteration: 1,
        chosen_skill: 'refactor',
        heaviest_dimension: 'standards',
        gaming_cause: 'gamed:test-weakening,suppression',
      },
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={gamed} />);
    expect(html).toContain('gamed:test-weakening,suppression');
    expect(html).toContain('flagged for review');
  });

  it('renders a flagged-for-review iteration without the gamed wording', () => {
    const flagged: DecisionTraceEntry[] = [
      { iteration: 1, chosen_skill: 'test', heaviest_dimension: 'tests', gaming_cause: 'flagged-for-review' },
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={flagged} />);
    expect(html).toContain('flagged-for-review');
    expect(html).not.toContain('gamed:');
  });

  it('renders an empty state when there are no entries', () => {
    const html = renderToStaticMarkup(<DecisionTraceList entries={[]} />);
    expect(html).toContain('No controller decisions recorded yet');
  });
});
