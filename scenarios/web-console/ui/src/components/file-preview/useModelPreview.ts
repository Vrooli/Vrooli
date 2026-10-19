import { useEffect, useRef, useState, type RefObject } from "react";
import { MAX_MODEL_BYTES, modelError, type ModelError, type ModelFormat, type ModelStats, type PreparedModel } from "./modelData";
import type { ModelStage } from "./modelStage";

export function useModelPreview(container: RefObject<HTMLDivElement>, href: string, format: ModelFormat | null, size: number, attempt: number) {
  const [stats, setStats] = useState<ModelStats | null>(null), [error, setError] = useState<ModelError | null>(null), [partial, setPartial] = useState(false);
  const stage = useRef<ModelStage | null>(null);
  useEffect(() => {
    setStats(null); setError(null); setPartial(false);
    if (size > MAX_MODEL_BYTES || !format) { setError(format ? "limit" : "unsupported"); return; }
    let worker: Worker | undefined;
    const controller = new AbortController();
    const canceled = () => controller.signal.aborted;
    const stop = () => { controller.abort(); stage.current?.dispose(); stage.current = null; worker?.terminate(); };
    const fail = (reason: ModelError) => { if (canceled()) return; clearTimeout(deadline); stop(); setStats(null); setError(reason); };
    let deadline = setTimeout(() => { fail("fetch"); }, 45000);
    try {
      worker = new Worker(new URL("./model.worker.ts", import.meta.url), { type: "module" });
      worker.onerror = () => { fail("unavailable"); };
      worker.onmessage = (event: MessageEvent<{ type: "ready"; data: PreparedModel } | { type: "error"; error: ModelError } | { type: "parsing" }>) => {
        if (canceled()) return;
        if (event.data.type === "parsing") { clearTimeout(deadline); deadline = setTimeout(() => { fail("limit"); }, 10000); return; }
        worker?.terminate();
        if (event.data.type === "error") { fail(event.data.error); return; }
        clearTimeout(deadline); deadline = setTimeout(() => { fail("unavailable"); }, 20000);
        const data = event.data.data;
        setPartial("partial" in data && data.partial);
        void import("./modelStage").then(async module => {
          if (canceled() || !container.current) return;
          const result = await module.createModelStage(container.current, data, controller.signal, value => { if (!canceled()) setStats(value); }, () => { fail("context"); });
          if (canceled()) { result?.dispose(); return; }
          stage.current = result; clearTimeout(deadline);
        }).catch((cause: unknown) => { fail(modelError(cause)); });
      };
      worker.postMessage({ href, format });
    } catch { fail("unavailable"); }
    return () => { clearTimeout(deadline); stop(); };
  }, [container, href, format, size, attempt]);
  return { stats, error, partial, stage };
}
