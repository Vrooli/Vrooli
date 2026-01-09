import { useState, useEffect } from "react";
import { BrowserRouter, Routes, Route, useNavigate, useParams, useLocation } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { TooltipProvider } from "./components/ui/tooltip";
import { ToastProvider, useToast } from "./components/ui/toast";
import { CampaignList } from "./components/CampaignList";
import { CampaignDetail } from "./components/CampaignDetail";
import { CreateCampaignDialog } from "./components/CreateCampaignDialog";
import { KeyboardShortcutsDialog } from "./components/KeyboardShortcutsDialog";
import { createCampaign } from "./lib/api";
import type { CreateCampaignRequest } from "./lib/api";

function CampaignListPage() {
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [helpDialogOpen, setHelpDialogOpen] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const createMutation = useMutation({
    mutationFn: createCampaign,
    onSuccess: (campaign) => {
      queryClient.invalidateQueries({ queryKey: ["campaigns"] });
      setCreateDialogOpen(false);
      navigate(`/campaign/${campaign.id}`);
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
      if (e.key === '?' && location.pathname === '/') {
        const target = e.target as HTMLElement;
        if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA') {
          e.preventDefault();
          setHelpDialogOpen(true);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [location]);

  const handleViewCampaign = (id: string) => {
    navigate(`/campaign/${id}`);
  };

  const handleCreateCampaign = (data: CreateCampaignRequest) => {
    createMutation.mutate(data);
  };

  return (
    <>
      <CampaignList
        onViewCampaign={handleViewCampaign}
        onCreateClick={() => setCreateDialogOpen(true)}
        onHelpClick={() => setHelpDialogOpen(true)}
      />

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
    </>
  );
}

function CampaignDetailPage() {
  const { campaignId } = useParams<{ campaignId: string }>();
  const navigate = useNavigate();

  const handleBackToList = () => {
    navigate('/');
  };

  if (!campaignId) {
    return null;
  }

  return <CampaignDetail campaignId={campaignId} onBack={handleBackToList} />;
}

function AppContent() {
  return (
    <TooltipProvider>
      <Routes>
        <Route path="/" element={<CampaignListPage />} />
        <Route path="/campaign/:campaignId" element={<CampaignDetailPage />} />
      </Routes>
    </TooltipProvider>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <ToastProvider>
        <AppContent />
      </ToastProvider>
    </BrowserRouter>
  );
}
