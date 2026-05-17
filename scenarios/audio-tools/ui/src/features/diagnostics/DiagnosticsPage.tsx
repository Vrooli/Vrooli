import { useRef, useState } from "react";
import type { ProviderTrace, SuiteCapability } from "../../services/diagnostics";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { SuiteCard } from "./SuiteCard";
import { TranscribeTryIt } from "./TranscribeTryIt";
import { SynthesizeTryIt } from "./SynthesizeTryIt";
import { SummarizeTryIt } from "./SummarizeTryIt";
import { TranscodeTryIt } from "./TranscodeTryIt";
import { ProviderTraceCard, type TraceEntry, type TraceFilter } from "./ProviderTraceCard";

// DiagnosticsPage — dashboard layout.
//   Top : SuiteCard summary strip (sticky on the page background).
//   Body: 2-col responsive grid — capability panels (left/wide) + provider
//         trace rail (right, sticky on lg+, stacks below on mobile).
//   Tile clicks scroll to the matching panel and, on failure, narrow the
//   trace rail to that capability.
export function DiagnosticsPage() {
  const { t } = useTranslation();
  const [trace, setTrace] = useState<TraceEntry[]>([]);
  const [filter, setFilter] = useState<TraceFilter>("all");

  const recordTrace = (capability: string, tr: ProviderTrace) =>
    setTrace((prev) => [{ ...tr, capability, emittedAt: Date.now() }, ...prev].slice(0, 20));

  const panelRefs: Record<SuiteCapability, React.RefObject<HTMLDivElement>> = {
    stt: useRef<HTMLDivElement>(null),
    tts: useRef<HTMLDivElement>(null),
    summarize: useRef<HTMLDivElement>(null),
    transcode: useRef<HTMLDivElement>(null),
  };

  const handleTileClick = (capability: SuiteCapability, failed: boolean) => {
    panelRefs[capability].current?.scrollIntoView({ behavior: "smooth", block: "start" });
    if (failed) setFilter(capability);
  };

  return (
    <div className="flex max-w-7xl flex-col gap-4">
      <header className="sticky top-0 z-10 -mx-2 border-b border-app-border bg-app-background/95 px-2 py-3 backdrop-blur supports-[backdrop-filter]:bg-app-background/75 md:-mx-4 md:px-4">
        <div className="flex flex-wrap items-end justify-between gap-2">
          <div className="min-w-0">
            <h1 className="truncate text-xl font-semibold text-app-foreground md:text-2xl">
              {t(strings.diagnostics.title)}
            </h1>
            <p className="mt-0.5 text-sm text-app-muted-foreground">{t(strings.diagnostics.description)}</p>
          </div>
        </div>
      </header>

      <SuiteCard onTrace={recordTrace} onTileClick={handleTileClick} />

      <div className="grid gap-4 lg:grid-cols-[1fr,22rem]">
        <div className="grid gap-4 md:grid-cols-2">
          <div ref={panelRefs.stt} className="scroll-mt-24">
            <TranscribeTryIt onTrace={(tr) => recordTrace("stt", tr)} />
          </div>
          <div ref={panelRefs.tts} className="scroll-mt-24">
            <SynthesizeTryIt onTrace={(tr) => recordTrace("tts", tr)} />
          </div>
          <div ref={panelRefs.summarize} className="scroll-mt-24">
            <SummarizeTryIt onTrace={(tr) => recordTrace("summarize", tr)} />
          </div>
          <div ref={panelRefs.transcode} className="scroll-mt-24">
            <TranscodeTryIt />
          </div>
        </div>
        <ProviderTraceCard
          entries={trace}
          filter={filter}
          onFilterChange={setFilter}
          sticky
        />
      </div>
    </div>
  );
}
