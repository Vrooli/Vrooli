import { useState } from "react";
import { PageHeader } from "../../components/composites/PageHeader";
import type { ProviderTrace } from "../../services/diagnostics";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { SuiteCard } from "./SuiteCard";
import { TranscribeTryIt } from "./TranscribeTryIt";
import { SynthesizeTryIt } from "./SynthesizeTryIt";
import { SummarizeTryIt } from "./SummarizeTryIt";
import { TranscodeTryIt } from "./TranscodeTryIt";
import { ProviderTraceCard, type TraceEntry } from "./ProviderTraceCard";

// DiagnosticsPage — dashboard layout. SuiteCard owns the aggregate
// "everything works" signal; per-capability cards stack below for
// targeted try-it runs; the ProviderTraceCard right-rail consumes
// traces from both sources.
export function DiagnosticsPage() {
  const { t } = useTranslation();
  const [trace, setTrace] = useState<TraceEntry[]>([]);
  const recordTrace = (capability: string, tr: ProviderTrace) =>
    setTrace((prev) => [{ ...tr, capability, emittedAt: Date.now() }, ...prev].slice(0, 20));

  return (
    <div className="flex max-w-6xl flex-col gap-4 md:gap-6">
      <PageHeader title={t(strings.diagnostics.title)} description={t(strings.diagnostics.description)} />

      <div className="grid gap-4 md:grid-cols-[1fr,22rem]">
        <div className="flex flex-col gap-4">
          <SuiteCard onTrace={recordTrace} />
          <TranscribeTryIt onTrace={(tr) => recordTrace("stt", tr)} />
          <SynthesizeTryIt onTrace={(tr) => recordTrace("tts", tr)} />
          <SummarizeTryIt onTrace={(tr) => recordTrace("summarize", tr)} />
          <TranscodeTryIt />
        </div>
        <ProviderTraceCard entries={trace} />
      </div>
    </div>
  );
}
