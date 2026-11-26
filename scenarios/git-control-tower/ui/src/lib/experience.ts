/**
 * Optional bridge to the local experience agent. The agent is present only in
 * an opted-in local session, so every call is a no-op elsewhere.
 *
 * Use this only for state the DOM cannot express (a restore expectation, a
 * persisted preference). Never pass input values or user-authored text.
 */
type ExperienceSignal = (name: string, detail?: Record<string, string | number | boolean | null>) => void;

export function experienceSignal(name: string, detail?: Record<string, string | number | boolean | null>): void {
  try {
    const controller = (window as unknown as { __vrooliExperience?: { signal?: ExperienceSignal } }).__vrooliExperience;
    controller?.signal?.(name, detail);
  } catch {
    // Telemetry must never affect the UI.
  }
}
