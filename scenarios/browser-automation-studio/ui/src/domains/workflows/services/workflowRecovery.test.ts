import { beforeEach, describe, expect, it } from 'vitest';
import {
  beginWorkflowRecoveryOperation,
  completeWorkflowRecoveryOperation,
  resetWorkflowRecoveryOperations,
} from './workflowRecovery';

describe('workflow recovery operation IDs', () => {
  beforeEach(resetWorkflowRecoveryOperations);

  it('reuses an ID only for the identical logical request payload', () => {
    const first = beginWorkflowRecoveryOperation('create', {
      projectId: 'project-1', name: 'Flow', aiPrompt: 'Visit the site',
    });
    const retry = beginWorkflowRecoveryOperation('create', {
      aiPrompt: 'Visit the site', name: 'Flow', projectId: 'project-1',
    });
    const edited = beginWorkflowRecoveryOperation('create', {
      projectId: 'project-1', name: 'Flow', aiPrompt: 'Visit the site and sign in',
    });

    expect(retry).toBe(first);
    expect(edited).not.toBe(first);
  });

  it('retires the ID after success so a later request is a new operation', () => {
    const first = beginWorkflowRecoveryOperation('modify', { workflowId: 'w1', prompt: 'Add wait' });
    completeWorkflowRecoveryOperation('modify', first);
    const next = beginWorkflowRecoveryOperation('modify', { workflowId: 'w1', prompt: 'Add wait' });

    expect(next).not.toBe(first);
  });
});
