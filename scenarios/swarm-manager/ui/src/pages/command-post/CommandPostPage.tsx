import { useCallback } from "react";
import { Menu, X } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { SummaryView } from "../../components/command-post/SummaryView";
import { ClarificationPanel } from "../../components/backlog/clarification-panel";
import { detailPath, decisionStreamPath, graphPath } from "../../app/routes/route-paths";
import { useAppBack } from "../../app/routes/useAppBack";
import { useEscapeRouteBack } from "../../app/routes/useEscapeRouteBack";
import { useAppShell } from "../../app/shell/AppShellContext";
import type { DetailRouteTarget } from "../../app/routes/route-paths";
import type { GraphLens } from "../../surfaces/graph/stores/graph-data-store";

export function CommandPostPage() {
  const navigate = useNavigate();
  const goBack = useAppBack(graphPath({ lens: "topology" }));
  const { openSidebar } = useAppShell();
  useEscapeRouteBack(goBack);

  const navigateToDetail = useCallback(
    (target: DetailRouteTarget) => {
      const path = detailPath(target);
      if (path) navigate(path);
    },
    [navigate],
  );

  const switchLens = useCallback(
    (lens: string) => {
      navigate(graphPath({ lens: lens as GraphLens }));
    },
    [navigate],
  );

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50" data-testid="command-post-page">
      <header className="sticky top-0 z-10 flex items-center gap-3 border-b border-slate-800 bg-slate-950/95 px-4 py-2.5 backdrop-blur-sm">
        <button
          type="button"
          onClick={openSidebar}
          className="rounded-lg p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Open sidebar"
          data-testid="page-sidebar-button"
        >
          <Menu className="h-5 w-5" />
        </button>
        <h1 className="min-w-0 flex-1 text-lg font-semibold text-slate-100">Command Post</h1>
        <button
          type="button"
          onClick={goBack}
          className="rounded-lg p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Close page"
          data-testid="command-post-back"
        >
          <X className="h-5 w-5" />
        </button>
      </header>

      <main className="mx-auto max-w-4xl px-4 py-3">
        <SummaryView
          onEnterDecisionStream={() => navigate(decisionStreamPath())}
          onNavigateToDetail={navigateToDetail}
          onSwitchLens={switchLens}
        />
      </main>

      <ClarificationPanel onAction={() => {}} />
    </div>
  );
}
