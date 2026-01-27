// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
// DOC: docs/guides/getting-started.md#ui-walkthrough
import { useMemo, useState } from "react";
import { FileSearch, Layers, Search, Sparkles, Type } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { SearchModeSelector } from "../../shared/components/SearchModeSelector";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import { consumeSearchIntent } from "../../shared/controllers/searchIntent";
import { DEFAULT_SEARCH_MODE, type SearchMode } from "../../shared/controllers/searchModes";
import type { Route } from "../../shared/controllers/routeController";
import { DocSearchPanelContainer } from "./DocSearchPanelContainer";
import { DeepSearchPanelContainer } from "./DeepSearchPanelContainer";
import { SearchPanelContainer } from "./SearchPanelContainer";

export type SearchPageProps = {
  onNavigate: (route: Route) => void;
};

const MODE_CONFIG: Record<
  SearchMode,
  { title: string; description: string; icon: JSX.Element }
> = {
  semantic: {
    title: "Semantic Search",
    description: "Ask natural-language questions across the knowledge base.",
    icon: <Search className="h-5 w-5 ko-icon" />,
  },
  files: {
    title: "File Search",
    description: "Locate documentation by filename patterns or paths.",
    icon: <FileSearch className="h-5 w-5 ko-icon" />,
  },
  text: {
    title: "Text Search",
    description: "Search documentation content with regex support.",
    icon: <Type className="h-5 w-5 ko-icon" />,
  },
  unified: {
    title: "Unified Documentation Search",
    description: "Blend file, text, and semantic results together.",
    icon: <Layers className="h-5 w-5 ko-icon" />,
  },
  deep: {
    title: "Deep Documentation Search",
    description: "Spawn an agent to explore docs and follow references.",
    icon: <Sparkles className="h-5 w-5 ko-icon" />,
  },
};

export function SearchPage({ onNavigate }: SearchPageProps) {
  const [intent] = useState(() => consumeSearchIntent());
  const [mode, setMode] = useState<SearchMode>(() => intent?.mode ?? DEFAULT_SEARCH_MODE);

  const modeConfig = MODE_CONFIG[mode];
  const prefillValue = intent && intent.mode === mode ? intent.value : null;

  const panelContent = useMemo(() => {
    if (mode === "semantic") {
      return <SearchPanelContainer prefillQuery={prefillValue} autoRun={Boolean(prefillValue)} />;
    }
    if (mode === "deep") {
      return <DeepSearchPanelContainer prefillQuery={prefillValue} autoRun={Boolean(prefillValue)} />;
    }
    return (
      <DocSearchPanelContainer
        key={mode}
        mode={mode === "files" ? "files" : mode === "text" ? "text" : "unified"}
        prefillValue={prefillValue}
        autoRun={Boolean(prefillValue)}
      />
    );
  }, [mode, prefillValue]);

  return (
    <ErrorBoundary
      fallback={({ error, reset }) => (
        <PageShell>
          <SectionErrorState
            title="Search Panel Unavailable"
            description="The search UI encountered an unexpected error. You can retry or return to the dashboard."
            errorMessage={error.message}
            actions={[
              { label: "Retry Section", onClick: reset },
              { label: "Back to Dashboard", onClick: () => onNavigate("dashboard"), variant: "secondary" },
            ]}
          />
        </PageShell>
      )}
    >
      <PageShell>
        <Panel>
          <PanelHeader
            title={modeConfig.title}
            description={modeConfig.description}
            icon={modeConfig.icon}
            className="mb-4"
          />
          <SearchModeSelector
            mode={mode}
            onChange={setMode}
            showDescriptions
            testId={selectors.search.modeSelector}
          />
          <div className="mt-4">{panelContent}</div>
        </Panel>
      </PageShell>
    </ErrorBoundary>
  );
}
