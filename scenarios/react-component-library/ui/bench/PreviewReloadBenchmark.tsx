import { useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { I18nextProvider } from "react-i18next";

import { i18n, useTranslation } from "../src/i18n";
import { strings } from "../src/consts/strings";
import { FixtureFrame, type FixtureStatus } from "../src/workbench/component/FixtureFrame";

const SOURCE_BYTES = 50 * 1024;
const ASSET_ID = "react-component-library:ReloadBenchmark";

/** A valid, disposable component source with a deterministic 50 KiB UTF-8 size. */
function createBenchmarkSource(): string {
  const prefix = `/** @libraryId ${ASSET_ID}\n * @version 1.0.0\n */\nimport React from "react";\nconst padding = "`;
  const suffix =
    '";\nexport const ReloadBenchmark = () => React.createElement("div", { "data-testid": "synthetic-component" }, padding.slice(0, 32));\n';
  const payloadLength = SOURCE_BYTES - new TextEncoder().encode(prefix + suffix).length;
  if (payloadLength < 0) throw new Error("Synthetic component prefix exceeds 50 KiB");
  const source = `${prefix}${"x".repeat(payloadLength)}${suffix}`;
  if (new TextEncoder().encode(source).length !== SOURCE_BYTES) {
    throw new Error("Synthetic component fixture is not exactly 50 KiB");
  }
  return source;
}

declare global {
  interface Window {
    __rclReloadBenchResults?: Array<{ sequence: number; milliseconds: number }>;
    __rclReloadBenchFailure?: string;
  }
}

export function PreviewReloadBenchmark() {
  const { t } = useTranslation();
  const [revision, setRevision] = useState("initial");
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState<FixtureStatus>("loading");
  const sequence = useRef(0);
  const pending = useRef<{ sequence: number; startedAt: number } | null>(null);
  const source = useRef(createBenchmarkSource());

  const save = async () => {
    if (busy) return;
    setBusy(true);
    try {
      // The benchmark route acknowledges this PUT in memory and never writes
      // to the component library or a persistent API.
      const response = await fetch("/bench-save", { method: "PUT", body: source.current });
      if (!response.ok) throw new Error(`mock save returned HTTP ${response.status}`);
      const next = ++sequence.current;
      pending.current = { sequence: next, startedAt: performance.now() };
      setRevision(`synthetic.tsx:${next}`);
    } catch (error) {
      window.__rclReloadBenchFailure = error instanceof Error ? error.message : String(error);
    } finally {
      setBusy(false);
    }
  };

  const observeStatus = (next: FixtureStatus) => {
    setStatus(next);
    if (next === "error") {
      window.__rclReloadBenchFailure = "FixtureFrame entered error state";
      return;
    }
    if (next !== "ready" || !pending.current) return;

    const sample = pending.current;
    const promotionDeadline = performance.now() + 1000;
    const finishWhenPromoted = () => {
      const frame = document.querySelector<HTMLIFrameElement>('[data-testid="fixture-iframe"]');
      const promoted = frame?.style.visibility === "visible";
      if (!promoted && performance.now() < promotionDeadline) {
        requestAnimationFrame(finishWhenPromoted);
        return;
      }
      if (!promoted) {
        window.__rclReloadBenchFailure = "candidate reported ready but was not promoted";
        return;
      }
      pending.current = null;
      window.__rclReloadBenchResults ??= [];
      window.__rclReloadBenchResults.push({
        sequence: sample.sequence,
        milliseconds: performance.now() - sample.startedAt,
      });
    };
    requestAnimationFrame(finishWhenPromoted);
  };

  return (
    <main style={{ fontFamily: "sans-serif", padding: 16 }}>
      <h1>{t(strings.workbench.nav.preview)}</h1>
      <button type="button" data-testid="bench-save" disabled={busy} onClick={() => void save()}>
        {t(strings.workbench.component.save)}
      </button>
      <output data-testid="bench-status">
        {t(
          status === "error"
            ? strings.workbench.component.stateError
            : status === "loading"
              ? strings.workbench.component.stateLoading
              : status === "applied"
                ? strings.workbench.component.stateApplied
                : strings.workbench.component.stateReady,
        )}
      </output>
      <FixtureFrame
        assetId={ASSET_ID}
        storyId="default"
        version="1.0.0"
        sourceRevision={revision}
        height={240}
        title={t(strings.workbench.nav.preview)}
        onStatus={observeStatus}
      />
    </main>
  );
}

const container = document.getElementById("root");
if (!container) throw new Error("Benchmark root element is missing");

createRoot(container).render(
  <I18nextProvider i18n={i18n}>
    <PreviewReloadBenchmark />
  </I18nextProvider>,
);
