import { describe, expect, it } from 'vitest';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';

/**
 * EM-P6 anti-drift guard: the UI no longer speaks "phase" for steer skills (the
 * controller has no phases). This asserts none of the renamed phase identifiers
 * reappear in src/. The backend proto contract fields (`*_phase_index`,
 * `*_phase_iteration`, gap-metric `phase_loops`) are a genuinely-unrelated
 * concept and stay — the forbidden list is the specific skill-picker identifiers
 * only, so those backend fields never match.
 */
const FORBIDDEN = [
  'PhasePicker',
  'PhasePickerDialog',
  'getPhaseDisplayName',
  'formatPhaseName',
  'useMergedPhaseNames',
  'usePhaseUsage',
  'PhaseInfo',
  'BUILT_IN_PHASE',
  'prioritizeSelectedPhases',
  'isLoadingPhases',
  'phaseNames',
  'phaseId',
  'phaseIds',
];

const SRC = join(__dirname, '..');
const SELF = 'phase-vocabulary-guard.test.ts';

function walk(dir: string): string[] {
  const out: string[] = [];
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) {
      out.push(...walk(full));
    } else if (/\.(ts|tsx)$/.test(entry) && entry !== SELF) {
      out.push(full);
    }
  }
  return out;
}

describe('UI phase→skill vocabulary guard', () => {
  it('contains no residual phase-skill identifiers in src/', () => {
    const offenders: string[] = [];
    for (const file of walk(SRC)) {
      const content = readFileSync(file, 'utf8');
      for (const token of FORBIDDEN) {
        if (content.includes(token)) {
          offenders.push(`${file.replace(SRC, 'src')} → ${token}`);
        }
      }
    }
    expect(offenders, `residual phase identifiers found:\n${offenders.join('\n')}`).toEqual([]);
  });
});
