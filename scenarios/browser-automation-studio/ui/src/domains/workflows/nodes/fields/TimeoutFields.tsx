import { FC } from 'react';
import type { UseSyncedFieldResult } from '@hooks/useSyncedField';
import { NodeNumberField } from './NodeNumberField';
import { FieldRow } from './FieldRow';

export interface TimeoutFieldsProps {
  /** Timeout field state from useSyncedNumber */
  timeoutMs: UseSyncedFieldResult<number>;
  /** Wait field state from useSyncedNumber (optional) */
  waitForMs?: UseSyncedFieldResult<number>;
  /** Label for timeout field (default: "Timeout (ms)") */
  timeoutLabel?: string;
  /** Label for wait field (default: "Post-action wait (ms)") */
  waitLabel?: string;
  /** Minimum timeout value (default: 100) */
  minTimeout?: number;
  /** Minimum wait value (default: 0) */
  minWait?: number;
}

/**
 * Common timeout + wait fields pattern used across many node types.
 *
 * Renders as a FieldRow with timeout on the left and optional wait on the right.
 * If waitForMs is not provided, renders just the timeout field in a full-width layout.
 *
 * @example
 * ```tsx
 * const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 30000, {
 *   min: 100,
 *   fallback: 30000,
 *   onCommit: (v) => updateParams({ timeoutMs: v }),
 * });
 * const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
 *   min: 0,
 *   onCommit: (v) => updateParams({ waitForMs: v || undefined }),
 * });
 *
 * <TimeoutFields timeoutMs={timeoutMs} waitForMs={waitForMs} />
 *
 * // Custom labels
 * <TimeoutFields
 *   timeoutMs={timeoutMs}
 *   waitForMs={waitForMs}
 *   waitLabel="Post-click delay (ms)"
 * />
 *
 * // Timeout only
 * <TimeoutFields timeoutMs={timeoutMs} />
 * ```
 */
export const TimeoutFields: FC<TimeoutFieldsProps> = ({
  timeoutMs,
  waitForMs,
  timeoutLabel = 'Timeout (ms)',
  waitLabel = 'Post-action wait (ms)',
  minTimeout = 100,
  minWait = 0,
}) => {
  if (!waitForMs) {
    return <NodeNumberField field={timeoutMs} label={timeoutLabel} min={minTimeout} />;
  }

  return (
    <FieldRow>
      <NodeNumberField field={timeoutMs} label={timeoutLabel} min={minTimeout} />
      <NodeNumberField field={waitForMs} label={waitLabel} min={minWait} />
    </FieldRow>
  );
};

export default TimeoutFields;
