import { describe, expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { DecisionTraceList, EffectivenessTable } from './DecisionTracePanel';
import type { DecisionTraceEntry, EffectivenessRow } from '@/types/api';

describe('DecisionTraceList rendering', () => {
  it('renders the new per-iteration signals: token cost, findings flow, flags, halt', () => {
    const entries: DecisionTraceEntry[] = [
      {
        iteration: 1,
        chosen_skill: 'refactor',
        heaviest_dimension: 'standards',
        dimension_scores: { standards: 8 },
        score_before: 8,
        score_after: 12,
        realized_delta: -4,
        tokens_used: 2500,
        closed_by_dimension: { standards: 1 },
        introduced_by_dimension: { tests: 2 },
        regressed: true,
        veto_applied: true,
        halt_reason: 'thrashing_cycle',
        rationale: 'selected "refactor" by effectiveness',
      },
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={entries} />);

    expect(html).toContain('2500 tok');
    expect(html).toContain('closed 1');
    expect(html).toContain('introduced 2');
    expect(html).toContain('regressed');
    expect(html).toContain('regression veto');
    expect(html).toContain('halt: thrashing_cycle');
    expect(html).toContain('by effectiveness');
  });

  it('renders DTV fitness provenance: verdict, prior, exclusions, override', () => {
    const entries: DecisionTraceEntry[] = [
      {
        iteration: 1,
        chosen_skill: 'lint-fix',
        heaviest_dimension: 'standards',
        dtv_verdict: 'green',
        dtv_prior: 0.92,
        dtv_excluded: { refactor: 'dtv:red' },
        dtv_gate_override: false,
      },
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={entries} />);
    expect(html).toContain('DTV: green');
    expect(html).toContain('prior 0.92');
    expect(html).toContain('gated refactor (dtv:red)');
  });

  it('flags a degraded (fail-open) DTV selection', () => {
    const entries: DecisionTraceEntry[] = [
      { iteration: 1, chosen_skill: 'ux', heaviest_dimension: 'ui', dtv_verdict: 'unknown', dtv_degraded: true },
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={entries} />);
    expect(html).toContain('DTV degraded → P1');
  });
});

describe('EffectivenessTable rendering', () => {
  it('renders rows with net findings and efficacy from factory data', () => {
    const rows: EffectivenessRow[] = [
      {
        skill_id: 'lint-fix',
        dimension: 'standards',
        closed_count: 20,
        introduced_count: 0,
        net_closed: 20,
        total_runs: 5,
        total_tokens: 5000,
        avg_tokens_per_run: 1000,
        observed_efficacy_per_ktok: 3.33,
        expected_efficacy_per_ktok: 2.08,
      },
    ];
    const html = renderToStaticMarkup(<EffectivenessTable rows={rows} />);
    expect(html).toContain('lint-fix');
    expect(html).toContain('standards');
    expect(html).toContain('+20');
    expect(html).toContain('2.08/khtok');
  });

  it('renders an empty state when there is no data', () => {
    const html = renderToStaticMarkup(<EffectivenessTable rows={[]} />);
    expect(html).toContain('No effectiveness data yet');
  });
});
