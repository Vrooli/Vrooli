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

  it('renders predicted Δ alongside realized Δ and a calibration indicator (P4)', () => {
    const entries: DecisionTraceEntry[] = [
      {
        iteration: 1,
        chosen_skill: 'refactor',
        heaviest_dimension: 'standards',
        predicted_reduction: 6,
        realized_delta: 4,
      },
      {
        iteration: 2,
        chosen_skill: 'lint-fix',
        heaviest_dimension: 'standards',
        predicted_reduction: 3,
        realized_delta: 3,
      },
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={entries} />);
    expect(html).toContain('pred Δ −6.0');
    expect(html).toContain('Δ −4.0 (improved)');
    // MAE = (|6-4| + |3-3|)/2 = 1.0 over 2 iterations.
    expect(html).toContain('mean |predicted − realized| = 1.0 over 2 iterations');
  });

  it('omits the calibration indicator until an iteration has both predicted and realized', () => {
    const entries: DecisionTraceEntry[] = [
      { iteration: 1, chosen_skill: 'refactor', predicted_reduction: 6 }, // no realized yet
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={entries} />);
    expect(html).not.toContain('mean |predicted − realized|');
  });

  it('flags a gamed iteration (credit zeroed) and the maturity rung', () => {
    const gamed: DecisionTraceEntry[] = [
      {
        iteration: 1,
        chosen_skill: 'refactor',
        heaviest_dimension: 'standards',
        gaming_cause: 'gamed:test-weakening,suppression',
        current_rung: 'R1 Safe & standards-clean',
      },
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={gamed} />);
    expect(html).toContain('gamed:test-weakening,suppression');
    expect(html).toContain('credit zeroed');
    // The '&' in the rung label is HTML-escaped in static markup.
    expect(html).toContain('rung: R1 Safe');
    expect(html).toContain('standards-clean');
  });

  it('renders a flagged-for-review iteration without the gamed penalty wording', () => {
    const flagged: DecisionTraceEntry[] = [
      { iteration: 1, chosen_skill: 'test', heaviest_dimension: 'tests', gaming_cause: 'flagged-for-review' },
    ];
    const html = renderToStaticMarkup(<DecisionTraceList entries={flagged} />);
    expect(html).toContain('flagged-for-review');
    expect(html).not.toContain('credit zeroed');
  });

  it('prominently flags a proceed-cap-flag degraded gate with its cause (P2)', () => {
    const dtvUnavailable: DecisionTraceEntry[] = [
      { iteration: 1, chosen_skill: 'ux', heaviest_dimension: 'ui', gate_degraded_cause: 'dtv_unavailable' },
    ];
    const allRed: DecisionTraceEntry[] = [
      { iteration: 2, chosen_skill: 'refactor', heaviest_dimension: 'standards', gate_degraded_cause: 'all_red' },
    ];

    const unavailHtml = renderToStaticMarkup(<DecisionTraceList entries={dtvUnavailable} />);
    expect(unavailHtml).toContain('Degraded gate (DTV unavailable)');
    expect(unavailHtml).toContain('budget halved');
    expect(unavailHtml).toContain('role="alert"');

    const allRedHtml = renderToStaticMarkup(<DecisionTraceList entries={allRed} />);
    expect(allRedHtml).toContain('Degraded gate (all candidate skills DTV-red)');
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
