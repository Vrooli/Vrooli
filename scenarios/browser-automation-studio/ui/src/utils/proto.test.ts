import { describe, expect, it } from 'vitest';
import { ExecutionSchema } from '@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb';
import { parseProtoStrict } from './proto';

describe('parseProtoStrict', () => {
  it('parses valid protojson payloads with json field names', () => {
    // Note: Execution proto has json_name="id" for execution_id field
    const parsed = parseProtoStrict(ExecutionSchema, {
      id: 'exec-1',
      workflow_id: 'wf-1',
    });
    expect(parsed.executionId).toBe('exec-1');
    expect(parsed.workflowId).toBe('wf-1');
  });

  it('throws when unknown fields are present', () => {
    expect(() =>
      parseProtoStrict(ExecutionSchema, {
        id: 'exec-1',
        workflow_id: 'wf-1',
        extra_field: 'nope',
      }),
    ).toThrow();
  });
});
