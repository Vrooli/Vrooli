import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TabNav } from "./components/layout/TabNav";
import { DashboardPage } from "./pages/Dashboard";
import { RunsPage } from "./pages/Runs";
import { GeneratePage } from "./pages/Generate";
import { DocsPage } from "./pages/Docs";
import { useUIStore } from "./stores/uiStore";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30000,
      refetchOnWindowFocus: false
    }
  }
});

function AppContent() {
  const { activeTab, setActiveTab } = useUIStore();

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      <div className="mx-auto flex max-w-6xl flex-col gap-6 px-6 py-8">
        {/* Tab Navigation */}
        <nav className="rounded-2xl border border-white/10 bg-white/[0.02] p-4">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Test Genie</p>
              <p className="mt-1 text-sm text-slate-300">
                {activeTab === "dashboard" && "Quick actions and health overview"}
                {activeTab === "runs" && "Scenarios and test history"}
                {activeTab === "generate" && "AI-powered test generation"}
                {activeTab === "docs" && "Documentation browser"}
              </p>
            </div>
            <TabNav activeTab={activeTab} onTabChange={setActiveTab} />
          </div>
        </nav>

        {/* Page Content */}
        <main>
          {activeTab === "dashboard" && <DashboardPage />}
          {activeTab === "runs" && <RunsPage />}
          {activeTab === "generate" && <GeneratePage />}
          {activeTab === "docs" && <DocsPage />}
        </main>
      </div>
    </div>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppContent />
    </QueryClientProvider>
  );
}
