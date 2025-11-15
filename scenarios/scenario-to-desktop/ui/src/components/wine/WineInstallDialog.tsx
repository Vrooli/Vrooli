import { useState } from "react";
import { createPortal } from "react-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Button } from "../ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import {
  AlertCircle,
  CheckCircle,
  Loader2,
  Wine,
  X,
  Clock,
  Info,
  ExternalLink
} from "lucide-react";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface WineInstallMethod {
  id: string;
  name: string;
  description: string;
  requires_sudo: boolean;
  steps: string[];
  estimated_time: string;
}

interface WineCheckResponse {
  installed: boolean;
  version?: string;
  platform: string;
  required_for: string[];
  install_methods: WineInstallMethod[];
  recommended_method?: string;
}

interface WineInstallStatus {
  install_id: string;
  status: string;
  method: string;
  started_at: string;
  completed_at?: string;
  log: string[];
  error_log: string[];
}

interface WineInstallDialogProps {
  onClose: () => void;
  onInstallComplete: () => void;
}

export function WineInstallDialog({ onClose, onInstallComplete }: WineInstallDialogProps) {
  const queryClient = useQueryClient();
  const [selectedMethod, setSelectedMethod] = useState<string | null>(null);
  const [installId, setInstallId] = useState<string | null>(null);

  // Helper to render dialog content with Portal
  const renderDialog = (content: React.ReactNode) => {
    return createPortal(content, document.body);
  };

  // Check Wine status
  const { data: wineCheck, isLoading: checkingWine } = useQuery<WineCheckResponse>({
    queryKey: ['wine-check'],
    queryFn: async () => {
      const res = await fetch(buildUrl('/system/wine/check'));
      if (!res.ok) throw new Error('Failed to check Wine status');
      return res.json();
    }
  });

  // Install Wine mutation
  const installMutation = useMutation({
    mutationFn: async (method: string) => {
      const res = await fetch(buildUrl('/system/wine/install'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ method })
      });
      if (!res.ok) {
        const error = await res.json().catch(() => ({ message: res.statusText }));
        throw new Error(error.message || 'Failed to start installation');
      }
      return res.json();
    },
    onSuccess: (data) => {
      setInstallId(data.install_id);
    }
  });

  // Poll installation status
  const { data: installStatus } = useQuery<WineInstallStatus>({
    queryKey: ['wine-install-status', installId],
    queryFn: async () => {
      if (!installId) return null;
      const res = await fetch(buildUrl(`/system/wine/install/status/${installId}`));
      if (!res.ok) throw new Error('Failed to fetch install status');
      return res.json();
    },
    enabled: !!installId,
    refetchInterval: (data) => {
      // Stop polling if installation complete or failed
      if (data?.status === 'completed' || data?.status === 'failed') {
        return false;
      }
      return 2000; // Poll every 2 seconds during installation
    }
  });

  // When installation completes successfully, invalidate Wine check and notify parent
  if (installStatus && 'status' in installStatus && installStatus.status === 'completed' && installStatus.method === 'flatpak') {
    queryClient.invalidateQueries({ queryKey: ['wine-check'] });
    setTimeout(() => onInstallComplete(), 1000);
  }

  const handleInstall = (method: string) => {
    setSelectedMethod(method);
    installMutation.mutate(method);
  };

  if (checkingWine) {
    return renderDialog(
      <div className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm">
        <Card className="w-full max-w-2xl border-slate-700 bg-slate-900">
          <CardContent className="p-8 flex items-center justify-center gap-3">
            <Loader2 className="h-6 w-6 animate-spin text-blue-400" />
            <span className="text-slate-300">Checking Wine status...</span>
          </CardContent>
        </Card>
      </div>
    );
  }

  // If Wine is already installed, show success and close
  if (wineCheck?.installed) {
    return renderDialog(
      <div className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm">
        <Card className="w-full max-w-2xl border-green-700 bg-slate-900">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <CheckCircle className="h-8 w-8 text-green-400" />
                <div>
                  <CardTitle>Wine is Installed</CardTitle>
                  <CardDescription>{wineCheck.version}</CardDescription>
                </div>
              </div>
              <Button variant="outline" size="sm" onClick={onClose}>
                <X className="h-4 w-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <p className="text-slate-300 mb-4">
              Wine is already installed and ready for Windows builds.
            </p>
            <Button onClick={onClose} className="w-full">
              Continue
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  // Show installation progress
  if (installStatus) {
    const isInstalling = installStatus.status === 'installing';
    const isCompleted = installStatus.status === 'completed';
    const isFailed = installStatus.status === 'failed';

    return renderDialog(
      <div className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm">
        <Card className="w-full max-w-3xl border-slate-700 bg-slate-900">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                {isInstalling && <Loader2 className="h-8 w-8 animate-spin text-blue-400" />}
                {isCompleted && <CheckCircle className="h-8 w-8 text-green-400" />}
                {isFailed && <AlertCircle className="h-8 w-8 text-red-400" />}
                <div>
                  <CardTitle>
                    {isInstalling && "Installing Wine..."}
                    {isCompleted && "Installation Complete"}
                    {isFailed && "Installation Failed"}
                  </CardTitle>
                  <CardDescription>
                    {installStatus.method === 'flatpak' && "Installing Wine via Flatpak (no sudo required)"}
                    {installStatus.method === 'flatpak-auto' && "Auto-installing Flatpak + Wine (no sudo required)"}
                    {installStatus.method === 'appimage' && "Installing Wine AppImage (no sudo required)"}
                    {installStatus.method === 'skip' && "Skipping Windows build"}
                  </CardDescription>
                </div>
              </div>
              {!isInstalling && (
                <Button variant="outline" size="sm" onClick={onClose}>
                  <X className="h-4 w-4" />
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Installation Log */}
            <div className="max-h-64 overflow-y-auto rounded border border-slate-700 bg-slate-950 p-4 font-mono text-sm">
              {installStatus.log.map((line, i) => (
                <div key={i} className="text-slate-300">
                  {line}
                </div>
              ))}
              {installStatus.error_log.map((line, i) => (
                <div key={`error-${i}`} className="text-red-400">
                  {line}
                </div>
              ))}
            </div>

            {/* Action Buttons */}
            {isCompleted && installStatus.method === 'skip' && (
              <div className="flex gap-3">
                <Button onClick={onClose} className="flex-1">
                  Continue (Skip Windows)
                </Button>
              </div>
            )}

            {isCompleted && (installStatus.method === 'flatpak' || installStatus.method === 'flatpak-auto' || installStatus.method === 'appimage') && (
              <div className="flex gap-3">
                <Button onClick={onInstallComplete} className="flex-1">
                  Continue with Windows Build
                </Button>
              </div>
            )}

            {isFailed && (
              <div className="flex gap-3">
                <Button variant="outline" onClick={onClose} className="flex-1">
                  Cancel
                </Button>
                <Button
                  onClick={() => handleInstall('skip')}
                  className="flex-1"
                >
                  Skip Windows Build
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    );
  }

  // Show installation method selection
  return renderDialog(
    <div className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 overflow-y-auto">
      <div className="min-h-full flex items-center justify-center py-4">
        <Card className="w-full max-w-3xl border-slate-700 bg-slate-900 my-auto">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Wine className="h-8 w-8 text-blue-400" />
              <div>
                <CardTitle>Wine Required for Windows Builds</CardTitle>
                <CardDescription>
                  Building Windows executables on Linux requires Wine
                </CardDescription>
              </div>
            </div>
            <Button variant="outline" size="sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4 max-h-[70vh] overflow-y-auto">
          {/* Info Box */}
          <div className="flex gap-3 rounded border border-blue-700 bg-blue-950/20 p-4">
            <Info className="h-5 w-5 text-blue-400 flex-shrink-0 mt-0.5" />
            <div className="text-sm text-slate-300">
              <p className="mb-2">
                Wine is a compatibility layer that allows building Windows applications on Linux.
              </p>
              <p>
                We recommend the AppImage method - it's fastest and requires no dependencies.
              </p>
            </div>
          </div>

          {/* Installation Methods */}
          <div className="space-y-3 pb-4">
            {wineCheck?.install_methods && wineCheck.install_methods.length > 0 ? (
              wineCheck.install_methods.map((method) => (
                <Card
                  key={method.id}
                  className={`cursor-pointer border-2 transition-colors ${
                    selectedMethod === method.id
                      ? 'border-blue-500 bg-blue-950/20'
                      : 'border-slate-700 bg-slate-800/50 hover:border-slate-600'
                  }`}
                  onClick={() => setSelectedMethod(method.id)}
                >
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <CardTitle className="text-lg flex items-center gap-2">
                        {method.name}
                        {method.id === wineCheck?.recommended_method && (
                          <span className="text-xs bg-green-500/20 text-green-400 px-2 py-0.5 rounded">
                            Recommended
                          </span>
                        )}
                      </CardTitle>
                      <CardDescription className="mt-1">
                        {method.description}
                      </CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {/* Metadata */}
                    <div className="flex gap-4 text-sm">
                      <div className="flex items-center gap-2">
                        <Clock className="h-4 w-4 text-slate-400" />
                        <span className="text-slate-300">{method.estimated_time}</span>
                      </div>
                      {method.requires_sudo && (
                        <div className="flex items-center gap-2">
                          <AlertCircle className="h-4 w-4 text-yellow-400" />
                          <span className="text-yellow-400">Requires sudo</span>
                        </div>
                      )}
                      {!method.requires_sudo && (
                        <div className="flex items-center gap-2">
                          <CheckCircle className="h-4 w-4 text-green-400" />
                          <span className="text-green-400">No sudo required</span>
                        </div>
                      )}
                    </div>

                    {/* Steps */}
                    <div>
                      <div className="text-sm font-medium text-slate-300 mb-2">Steps:</div>
                      <ul className="space-y-1">
                        {method.steps.map((step, i) => {
                          // Check if step contains a URL
                          const urlMatch = step.match(/(https?:\/\/[^\s]+)/);
                          if (urlMatch) {
                            const [beforeUrl, url] = step.split(urlMatch[0]);
                            return (
                              <li key={i} className="text-sm text-slate-400 flex gap-2">
                                <span className="text-slate-600">{i + 1}.</span>
                                <span>
                                  {beforeUrl}
                                  <a
                                    href={url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-blue-400 hover:text-blue-300 inline-flex items-center gap-1"
                                  >
                                    {url}
                                    <ExternalLink className="h-3 w-3" />
                                  </a>
                                </span>
                              </li>
                            );
                          }
                          return (
                            <li key={i} className="text-sm text-slate-400 flex gap-2">
                              <span className="text-slate-600">{i + 1}.</span>
                              <span>{step}</span>
                            </li>
                          );
                        })}
                      </ul>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))
            ) : (
              <div className="text-center py-8 text-slate-400">
                <p>No installation methods available.</p>
                <p className="text-sm mt-2">Please check your system configuration.</p>
              </div>
            )}
          </div>

          {/* Action Buttons */}
          <div className="flex gap-3 pt-4">
            <Button variant="outline" onClick={onClose} className="flex-1">
              Cancel
            </Button>
            <Button
              onClick={() => selectedMethod && handleInstall(selectedMethod)}
              disabled={!selectedMethod || installMutation.isPending}
              className="flex-1"
            >
              {installMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Starting...
                </>
              ) : (
                'Continue'
              )}
            </Button>
          </div>

          {installMutation.isError && (
            <div className="flex items-center gap-2 text-red-400 text-sm">
              <AlertCircle className="h-4 w-4" />
              {(installMutation.error as Error).message}
            </div>
          )}
        </CardContent>
      </Card>
      </div>
    </div>
  );
}
