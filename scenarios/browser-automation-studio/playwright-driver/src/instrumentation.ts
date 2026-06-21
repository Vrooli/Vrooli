/**
 * Instrumentation Seam
 *
 * STABILITY: STABLE CORE (additive only)
 *
 * Cross-cutting telemetry hook for the driver, the TypeScript counterpart
 * to the Go-side `driver.TelemetryCollector`. It lets P2's performance
 * capture (CDP tracing, per-step timing) attach to the execution pipeline
 * WITHOUT the session manager or instruction executor knowing what is
 * being collected.
 *
 * P1 ships the seam threaded everywhere with a no-op default
 * (`noopInstrumentation`), so there is NO behavior change today. P2
 * supplies a real implementation that starts/stops tracing around a
 * session and records timing around each instruction.
 *
 * DESIGN:
 * - All hooks are optional on the interface so an implementation can
 *   subscribe to only the lifecycle points it cares about.
 * - Hooks are best-effort: callers invoke them defensively (see
 *   `safeInvoke`) so an instrumentation failure never breaks execution.
 * - Hooks may be async; callers await them. The no-op resolves instantly.
 *
 * @module instrumentation
 */

/**
 * Context passed to session-level instrumentation hooks.
 */
export interface SessionInstrumentationContext {
  /** Session identifier. */
  readonly sessionId: string;
  /** Execution identifier associated with the session, when known. */
  readonly executionId?: string;
}

/**
 * Context passed to per-instruction instrumentation hooks.
 */
export interface InstructionInstrumentationContext {
  /** Session identifier the instruction runs in. */
  readonly sessionId: string;
  /** Instruction type (e.g. "navigate", "click"). */
  readonly type: string;
  /** Step index within the workflow. */
  readonly index: number;
  /** Node identifier of the instruction. */
  readonly nodeId: string;
}

/**
 * Result of a per-instruction execution, handed to onInstructionEnd.
 */
export interface InstructionInstrumentationResult {
  /** Whether the instruction succeeded. */
  readonly success: boolean;
  /** Wall-clock duration of the handler execution, in milliseconds. */
  readonly durationMs: number;
  /** Error, when the instruction threw rather than returning a failure. */
  readonly error?: unknown;
}

/**
 * Instrumentation hooks. Every member is optional; an implementation
 * provides only the lifecycle points it needs.
 */
export interface Instrumentation {
  /** Invoked when a session becomes ready to accept instructions. */
  onSessionStart?(ctx: SessionInstrumentationContext): void | Promise<void>;
  /** Invoked when a session is closed/torn down. */
  onSessionClose?(ctx: SessionInstrumentationContext): void | Promise<void>;
  /** Invoked immediately before an instruction handler runs. */
  onInstructionStart?(ctx: InstructionInstrumentationContext): void | Promise<void>;
  /** Invoked immediately after an instruction handler completes (or throws). */
  onInstructionEnd?(
    ctx: InstructionInstrumentationContext,
    result: InstructionInstrumentationResult
  ): void | Promise<void>;
}

/**
 * The inert default instrumentation. Every hook is absent, so callers
 * short-circuit and do nothing. Shared singleton — safe to reuse.
 */
export const noopInstrumentation: Instrumentation = Object.freeze({});

/**
 * Resolve an optional instrumentation to a concrete value, defaulting to
 * the shared no-op when undefined.
 */
export function resolveInstrumentation(instr?: Instrumentation): Instrumentation {
  return instr ?? noopInstrumentation;
}

/**
 * Invoke an instrumentation hook defensively: a missing hook is a no-op,
 * and a throwing/rejecting hook is swallowed (instrumentation must never
 * break execution). Returns a promise that always resolves.
 */
export async function safeInvoke<T extends unknown[]>(
  hook: ((...args: T) => void | Promise<void>) | undefined,
  ...args: T
): Promise<void> {
  if (!hook) {
    return;
  }
  try {
    await hook(...args);
  } catch {
    // Instrumentation is best-effort; never propagate its failures.
  }
}
