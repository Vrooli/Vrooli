import type { VoiceCoreServices } from "./services";
import type { WakeWordEngine, WakeWordTemplate } from "./wakeword";

export interface WakeWordTemplateLoad {
  engine: WakeWordEngine | null;
  template: WakeWordTemplate | null;
}

/** Rebuild persisted raw-audio samples into the current engine's features. */
export async function loadWakeWordTemplate(
  services: Pick<VoiceCoreServices, "getWakeWordConfig" | "createWakeWordEngine" | "bytesToFeatures">,
  currentEngine: WakeWordEngine | null,
  threshold: number,
  onEngine: (engine: WakeWordEngine) => void,
): Promise<WakeWordTemplateLoad> {
  const config = await services.getWakeWordConfig();
  if (!config.configured || !config.template) return { engine: currentEngine, template: null };

  const engine = currentEngine ?? services.createWakeWordEngine();
  onEngine(engine);
  const samples = (
    await Promise.all(
      config.template.samples.map((sample) => services.bytesToFeatures(sample.audio, engine).catch(() => null)),
    )
  ).filter((features) => features !== null);
  if (samples.length === 0) return { engine, template: null };

  return {
    engine,
    template: {
      samples,
      label: config.template.label,
      threshold,
      updatedAt: config.template.updatedAt,
      calibration: engine.calibrate?.(samples) ?? null,
    },
  };
}
