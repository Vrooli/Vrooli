import { describe, expect, it } from 'vitest';
import {
  formatSkillSetLabel,
  formatSkillSetTooltip,
  getPhaseDisplayName,
  getQueueStepDisplay,
} from './utils';

const PHASES = [
  { id: 'progress', name: 'Progress' },
  { id: 'react-coherence', name: 'React Coherence' },
  { id: 'api-steer', name: 'API Steer' },
];

describe('steering label utils', () => {
  it('formats set labels with +k more', () => {
    expect(
      formatSkillSetLabel(['react-coherence', 'api-steer'], PHASES, { maxVisible: 1 })
    ).toBe('React Coherence +1 more');
  });

  it('formats set labels without truncation when visible count covers all', () => {
    expect(
      formatSkillSetLabel(['react-coherence', 'api-steer'], PHASES, { maxVisible: 2 })
    ).toBe('React Coherence, API Steer');
  });

  it('returns empty label when set is empty', () => {
    expect(formatSkillSetLabel([], PHASES, { emptyLabel: 'None selected' })).toBe('None selected');
  });

  it('builds full tooltip labels', () => {
    expect(formatSkillSetTooltip(['react-coherence', 'api-steer'], PHASES)).toBe(
      'React Coherence, API Steer'
    );
  });

  it('returns queue step summary and tooltip', () => {
    const result = getQueueStepDisplay(['react-coherence', 'api-steer'], PHASES);
    expect(result.label).toBe('React Coherence +1 more');
    expect(result.tooltip).toBe('React Coherence, API Steer');
  });

  it('falls back to phase formatting for unknown ids', () => {
    expect(getPhaseDisplayName('custom-skill', PHASES)).toBe('Custom Skill');
  });
});
