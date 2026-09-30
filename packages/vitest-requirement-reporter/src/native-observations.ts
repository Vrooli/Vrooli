import type { File, Task } from 'vitest';

// Exact version: the retained native conformance fixture covers 2.1.9 only.
export const supportedNativeVersion = '2.1.9';
export function nativeAssertionOptions(version: string): { requireAssertions?: true } {
  return version === supportedNativeVersion ? { requireAssertions: true } : {};
}
export const nativeLimitations = [
  'Assertion activity does not prove matcher completion or behavioral adequacy.',
  'Bare expect calls, tautologies, and setup-hook assertions can satisfy the native check.',
  'Alternate assertion libraries are not observed by the native expect check.',
  'Retry results expose final state and retained errors, not a complete attempt history.',
] as const;

export interface NativeObservation {
  test_id: string;
  file: string;
  project: string;
  name: string;
  state: string;
  // Final aggregate count, not an observed attempt ordinal/history.
  retry_count?: number;
  seed?: string;
  errors: string[];
  require_assertions: boolean;
  assertion_status: 'checked_clean' | 'violation' | 'unknown';
  reason: string;
  // Declared links from the existing reporter grammar, never proof of behavior.
  // Absent means no tag extractor was supplied, not an observed empty link set.
  requirement_ids?: string[];
}

export interface NativeObservationReport {
  run_id: string;
  schema_version: 'vitest-native/v1';
  runner_version: string;
  support_profile: string;
  limitations: readonly string[];
  unhandled_errors: string[];
  tests: NativeObservation[];
}

export function nativeObservations(
  files: File[], version: string,
  requireAssertions: (project: string) => boolean,
  unhandledErrors: readonly unknown[] = [], runID = '',
  requirementIDs?: (task: Task) => string[],
  projectSeed?: (project: string) => number | undefined,
): NativeObservationReport {
  const tests: NativeObservation[] = [];
  const message = (error: unknown): string => {
    if (error && typeof error === 'object' && 'message' in error) return String(error.message);
    return String(error);
  };
  for (const file of files) {
    const project = file.projectName ?? '';
    const enabled = requireAssertions(project);
    const visit = (task: Task): void => {
      if (task.type === 'suite') { task.tasks.forEach(visit); return; }
      const state = task.result?.state ?? task.mode ?? 'unknown';
      const errors = (task.result?.errors ?? []).map(message);
      let assertion_status: NativeObservation['assertion_status'] = 'unknown';
      let reason = 'not-executed';
      if (version !== supportedNativeVersion) reason = 'unsupported-version';
      else if (!enabled) reason = 'missing-input';
      else if (state === 'skip' || state === 'todo') reason = 'skipped';
      else if (task.fails) reason = 'unsupported-test-kind';
      else if (state === 'pass') { assertion_status = 'checked_clean'; reason = 'none'; }
      else if (state === 'fail') {
        reason = 'missing-input'; // A setup or matcher failure does not establish assertion absence.
        if (errors.some(error => error === 'expected any number of assertion, but got none')) {
          assertion_status = 'violation'; reason = 'none';
        }
      }
      tests.push({ test_id: task.id, file: file.filepath, project, name: task.name,
        state, retry_count: task.result?.retryCount, errors,
        seed: projectSeed?.(project)?.toString(),
        require_assertions: enabled, assertion_status, reason,
        requirement_ids: requirementIDs?.(task) });
    };
    file.tasks.forEach(visit);
  }
  return { run_id: runID, schema_version: 'vitest-native/v1', runner_version: version,
    support_profile: version === supportedNativeVersion ? 'react-vitest-v2' : 'unsupported',
    limitations: [...nativeLimitations], unhandled_errors: unhandledErrors.map(message), tests };
}
