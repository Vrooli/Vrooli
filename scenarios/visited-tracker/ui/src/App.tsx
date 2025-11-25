import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { TooltipProvider } from "./components/ui/tooltip";
import { CampaignList } from "./components/CampaignList";
import { CampaignDetail } from "./components/CampaignDetail";
import { CreateCampaignDialog } from "./components/CreateCampaignDialog";
import { createCampaign } from "./lib/api";
import type { CreateCampaignRequest } from "./lib/api";

type View = "list" | "detail";

export default function App() {
  const [currentView, setCurrentView] = useState<View>("list");
  const [selectedCampaignId, setSelectedCampaignId] = useState<string | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: createCampaign,
    onSuccess: (campaign) => {
      queryClient.invalidateQueries({ queryKey: ["campaigns"] });
      setCreateDialogOpen(false);
      setSelectedCampaignId(campaign.id);
      setCurrentView("detail");
    },
    onError: (error) => {
      alert(`Failed to create campaign: ${(error as Error).message}`);
    }
  });

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
    </TooltipProvider>
  );
}
