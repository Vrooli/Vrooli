import { useState, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { TooltipProvider } from "./components/ui/tooltip";
import { ToastProvider, useToast } from "./components/ui/toast";
import { CampaignList } from "./components/CampaignList";
import { CampaignDetail } from "./components/CampaignDetail";
import { CreateCampaignDialog } from "./components/CreateCampaignDialog";
import { KeyboardShortcutsDialog } from "./components/KeyboardShortcutsDialog";
import { createCampaign } from "./lib/api";
import type { CreateCampaignRequest } from "./lib/api";

type View = "list" | "detail";

function AppContent() {
  const [currentView, setCurrentView] = useState<View>("list");
  const [selectedCampaignId, setSelectedCampaignId] = useState<string | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [helpDialogOpen, setHelpDialogOpen] = useState(false);

  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const createMutation = useMutation({
    mutationFn: createCampaign,
    onSuccess: (campaign) => {
      queryClient.invalidateQueries({ queryKey: ["campaigns"] });
      setCreateDialogOpen(false);
      setSelectedCampaignId(campaign.id);
      setCurrentView("detail");
      showToast(`Campaign "${campaign.name}" created successfully`, "success");
    },
    onError: (error) => {
      showToast(`Failed to create campaign: ${(error as Error).message}`, "error");
    }
  });

  // Global keyboard shortcuts for better UX
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // ? to show help modal
      if (e.key === '?' && currentView === "list") {
        const target = e.target as HTMLElement;
        if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA') {
          e.preventDefault();
          setHelpDialogOpen(true);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [currentView]);

  const handleViewCampaign = (id: string) => {
    setSelectedCampaignId(id);
    setCurrentView("detail");
  };

  const handleBackToList = () => {
    setCurrentView("list");
    setSelectedCampaignId(null);
  };

  const handleCreateCampaign = (data: CreateCampaignRequest) => {
    createMutation.mutate(data);
  };

  return (
    <TooltipProvider>
      {currentView === "list" ? (
        <CampaignList
          onViewCampaign={handleViewCampaign}
          onCreateClick={() => setCreateDialogOpen(true)}
          onHelpClick={() => setHelpDialogOpen(true)}
        />
      ) : selectedCampaignId ? (
        <CampaignDetail campaignId={selectedCampaignId} onBack={handleBackToList} />
      ) : null}

      <CreateCampaignDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onSubmit={handleCreateCampaign}
        isLoading={createMutation.isPending}
      />

      <KeyboardShortcutsDialog
        open={helpDialogOpen}
        onOpenChange={setHelpDialogOpen}
      />
    </TooltipProvider>
  );
}

export default function App() {
  return (
    <ToastProvider>
      <AppContent />
    </ToastProvider>
  );
}
