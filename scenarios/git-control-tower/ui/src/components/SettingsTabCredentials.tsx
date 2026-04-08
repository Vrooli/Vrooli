import { useState, useEffect } from "react";
import {
  CheckCircle,
  AlertCircle,
  Eye,
  EyeOff,
  ExternalLink,
  RefreshCw,
  Lock,
} from "lucide-react";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import {
  useCredentials,
  useSaveCredential,
  useTestCredential,
  useUpdateRemoteURL,
} from "../lib/hooks";
import { testCredential } from "../lib/api";
import { convertSSHToHTTPS, convertHTTPSToSSH, detectUrlType } from "../lib/gitUrlUtils";
import { SettingsTabCredentialsSSH } from "./SettingsTabCredentialsSSH";

interface SettingsTabCredentialsProps {
  remoteUrl?: string;
  hasUpstream?: boolean;
  isMobile: boolean;
  repoId?: string | null;
}

export function SettingsTabCredentials({
  remoteUrl,
  hasUpstream,
  isMobile,
  repoId
}: SettingsTabCredentialsProps) {
  const credentialsQuery = useCredentials(repoId);
  const saveMutation = useSaveCredential(repoId);
  const testMutation = useTestCredential(repoId);
  const updateUrlMutation = useUpdateRemoteURL(repoId);

  const [username, setUsername] = useState("");
  const [token, setToken] = useState("");
  const [showToken, setShowToken] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [testResult, setTestResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);

  // Live auth status from auto-test on mount
  const [authStatus, setAuthStatus] = useState<{
    tested: boolean;
    loading: boolean;
    success: boolean;
    authorized: boolean;
    reachable: boolean;
    error?: string;
  }>({ tested: false, loading: false, success: false, authorized: false, reachable: false });

  const urlType = remoteUrl ? detectUrlType(remoteUrl) : "https";
  const isSSH = urlType === "ssh";

  // Get the credential for origin remote
  const originCredential = credentialsQuery.data?.credentials?.find(
    (c) => c.remote === "origin"
  );

  // Pre-fill username from stored credential
  useEffect(() => {
    if (originCredential?.username && !username) {
      setUsername(originCredential.username);
    }
  }, [originCredential?.username, username]);

  // Clear success message after 3 seconds
  useEffect(() => {
    if (saveSuccess) {
      const timer = setTimeout(() => setSaveSuccess(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [saveSuccess]);

  // Clear test result after 5 seconds
  useEffect(() => {
    if (testResult) {
      const timer = setTimeout(() => setTestResult(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [testResult]);

  // Auto-test auth status on mount and when credentials change
  useEffect(() => {
    if (!remoteUrl || !repoId) return;
    let cancelled = false;
    setAuthStatus(prev => ({ ...prev, loading: true }));
    testCredential({ remote: "origin", use_stored: true }, repoId)
      .then(result => {
        if (cancelled) return;
        setAuthStatus({
          tested: true,
          loading: false,
          success: result.success,
          authorized: result.authorized,
          reachable: result.reachable,
          error: result.error,
        });
      })
      .catch(() => {
        if (cancelled) return;
        setAuthStatus({
          tested: true,
          loading: false,
          success: false,
          authorized: false,
          reachable: false,
          error: "Failed to test connection",
        });
      });
    return () => { cancelled = true; };
  }, [remoteUrl, repoId, credentialsQuery.dataUpdatedAt]);

  const handleSave = async () => {
    if (!username.trim() || !token.trim()) return;

    const result = await saveMutation.mutateAsync({
      remote: "origin",
      username: username.trim(),
      token: token.trim()
    });

    if (result.success) {
      setToken(""); // Clear token after save
      setSaveSuccess(true);
      credentialsQuery.refetch();
    }
  };

  const handleTest = async () => {
    setTestResult(null);

    const result = await testMutation.mutateAsync({
      remote: "origin",
      use_stored: true
    });

    setTestResult({
      success: result.success,
      message: result.success
        ? "Connection successful!"
        : result.error || "Connection failed"
    });

    // Also update the live auth status
    setAuthStatus({
      tested: true,
      loading: false,
      success: result.success,
      authorized: result.authorized,
      reachable: result.reachable,
      error: result.error,
    });
  };

  const handleSwitchProtocol = async () => {
    if (!remoteUrl) return;

    const newUrl = isSSH
      ? convertSSHToHTTPS(remoteUrl)
      : convertHTTPSToSSH(remoteUrl);

    await updateUrlMutation.mutateAsync({
      remote: "origin",
      url: newUrl
    });
  };

  if (!remoteUrl || !hasUpstream) {
    return (
      <div className="text-center py-8">
        <AlertCircle className="h-8 w-8 text-slate-500 mx-auto mb-3" />
        <p className="text-sm text-slate-400 mb-2">No remote configured</p>
        <p className="text-xs text-slate-500">
          Add an upstream remote to configure credentials.
        </p>
      </div>
    );
  }

  const inputClasses = isMobile
    ? "w-full rounded-xl border border-slate-700 bg-slate-900/60 px-4 py-3 text-sm text-slate-100 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
    : "w-full rounded-lg border border-slate-700 bg-slate-900/60 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500";

  const buttonHeight = isMobile ? "h-11" : "h-8";

  return (
    <div className={isMobile ? "space-y-6" : "space-y-5"}>
      {/* Remote URL Section */}
      <div className={isMobile ? "space-y-3" : "space-y-2"}>
        <h3 className={`font-semibold text-slate-200 ${isMobile ? "text-sm" : "text-xs"}`}>
          Remote URL
        </h3>
        <div className={`flex items-center gap-2 ${isMobile ? "flex-col" : "flex-row"}`}>
          <div className={`flex-1 ${isMobile ? "w-full" : ""}`}>
            <div className={`flex items-center gap-2 rounded-lg border border-slate-700 bg-slate-900/40 ${isMobile ? "px-4 py-3" : "px-3 py-2"}`}>
              <Badge variant={isSSH ? "info" : "staged"} className="text-[10px] shrink-0">
                {isSSH ? "SSH" : "HTTPS"}
              </Badge>
              <span className={`text-slate-300 truncate ${isMobile ? "text-sm" : "text-xs"}`} title={remoteUrl}>
                {remoteUrl}
              </span>
            </div>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={handleSwitchProtocol}
            disabled={updateUrlMutation.isPending}
            className={`${buttonHeight} ${isMobile ? "w-full" : "shrink-0"}`}
          >
            {updateUrlMutation.isPending ? (
              <RefreshCw className="h-3 w-3 animate-spin mr-2" />
            ) : null}
            Switch to {isSSH ? "HTTPS" : "SSH"}
          </Button>
        </div>
      </div>

      {/* Authentication Status */}
      <div className={isMobile ? "space-y-3" : "space-y-2"}>
        <h3 className={`font-semibold text-slate-200 ${isMobile ? "text-sm" : "text-xs"}`}>
          Authentication Status
        </h3>
        <div className={`flex items-center gap-2 rounded-lg border ${
          authStatus.loading
            ? "border-slate-700 bg-slate-900/40"
            : authStatus.success
              ? "border-emerald-700/50 bg-emerald-950/20"
              : authStatus.tested && authStatus.reachable
                ? "border-red-700/50 bg-red-950/20"
                : authStatus.tested
                  ? "border-amber-700/50 bg-amber-950/20"
                  : "border-slate-700 bg-slate-900/40"
        } ${isMobile ? "px-4 py-3" : "px-3 py-2"}`}>
          {authStatus.loading ? (
            <>
              <RefreshCw className="h-4 w-4 text-slate-400 animate-spin shrink-0" />
              <span className={`text-slate-300 ${isMobile ? "text-sm" : "text-xs"}`}>
                Checking authentication...
              </span>
            </>
          ) : authStatus.success ? (
            <>
              <CheckCircle className="h-4 w-4 text-emerald-400 shrink-0" />
              <span className={`text-emerald-200 ${isMobile ? "text-sm" : "text-xs"}`}>
                Authenticated
                {originCredential?.type === "ssh" && originCredential.ssh_key_path ? (
                  <span className="text-emerald-400/70 ml-1">
                    (SSH key: {originCredential.ssh_key_path.split("/").pop()})
                  </span>
                ) : originCredential?.is_configured && originCredential.token_masked ? (
                  <span className="text-emerald-400/70 ml-1">
                    ({originCredential.token_masked})
                  </span>
                ) : isSSH ? (
                  <span className="text-emerald-400/70 ml-1">
                    (via system SSH)
                  </span>
                ) : null}
              </span>
            </>
          ) : authStatus.tested && authStatus.reachable ? (
            <>
              <AlertCircle className="h-4 w-4 text-red-400 shrink-0" />
              <span className={`text-red-200 ${isMobile ? "text-sm" : "text-xs"}`}>
                Authentication failed — check credentials
              </span>
            </>
          ) : authStatus.tested ? (
            <>
              <AlertCircle className="h-4 w-4 text-amber-400 shrink-0" />
              <span className={`text-amber-200 ${isMobile ? "text-sm" : "text-xs"}`}>
                {authStatus.error || "Cannot reach remote"}
              </span>
            </>
          ) : (
            <>
              <AlertCircle className="h-4 w-4 text-slate-500 shrink-0" />
              <span className={`text-slate-400 ${isMobile ? "text-sm" : "text-xs"}`}>
                Not tested yet
              </span>
            </>
          )}
        </div>
      </div>

      {/* Credential Setup (HTTPS only) */}
      {!isSSH && (
        <div className={isMobile ? "space-y-4" : "space-y-3"}>
          <h3 className={`font-semibold text-slate-200 ${isMobile ? "text-sm" : "text-xs"}`}>
            {originCredential?.is_configured ? "Update Credentials" : "Configure Credentials"}
          </h3>

          <div className={isMobile ? "space-y-4" : "space-y-3"}>
            <div className={isMobile ? "space-y-2" : "space-y-1"}>
              <label className={`text-slate-400 ${isMobile ? "text-sm" : "text-[11px]"}`}>
                Username
              </label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="GitHub username"
                className={inputClasses}
                autoComplete="username"
              />
            </div>

            <div className={isMobile ? "space-y-2" : "space-y-1"}>
              <label className={`text-slate-400 ${isMobile ? "text-sm" : "text-[11px]"}`}>
                Personal Access Token
              </label>
              <div className="relative">
                <input
                  type={showToken ? "text" : "password"}
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  placeholder="ghp_xxxxxxxxxxxx"
                  className={`${inputClasses} pr-10`}
                  autoComplete="current-password"
                />
                <button
                  type="button"
                  onClick={() => setShowToken(!showToken)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-slate-500 hover:text-slate-300"
                  aria-label={showToken ? "Hide token" : "Show token"}
                >
                  {showToken ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              </div>
              <a
                href="https://github.com/settings/tokens/new?description=git-control-tower&scopes=repo"
                target="_blank"
                rel="noopener noreferrer"
                className={`inline-flex items-center gap-1 text-blue-400 hover:text-blue-300 ${isMobile ? "text-xs mt-1" : "text-[11px]"}`}
              >
                <ExternalLink className="h-3 w-3" />
                Create a GitHub token
              </a>
            </div>

            <div className={`flex gap-2 ${isMobile ? "flex-col" : "flex-row"}`}>
              <Button
                variant="default"
                size="sm"
                onClick={handleSave}
                disabled={!username.trim() || !token.trim() || saveMutation.isPending}
                className={`${buttonHeight} ${isMobile ? "w-full" : ""}`}
              >
                {saveMutation.isPending ? (
                  <RefreshCw className="h-3 w-3 animate-spin mr-2" />
                ) : (
                  <Lock className="h-3 w-3 mr-2" />
                )}
                Save Credentials
              </Button>

              <Button
                variant="outline"
                size="sm"
                onClick={handleTest}
                disabled={testMutation.isPending}
                className={`${buttonHeight} ${isMobile ? "w-full" : ""}`}
              >
                {testMutation.isPending ? (
                  <RefreshCw className="h-3 w-3 animate-spin mr-2" />
                ) : null}
                Test Connection
              </Button>
            </div>

            {/* Feedback Messages */}
            {saveSuccess && (
              <div className="flex items-center gap-2 text-emerald-400">
                <CheckCircle className="h-4 w-4" />
                <span className={isMobile ? "text-sm" : "text-xs"}>
                  Credentials saved successfully
                </span>
              </div>
            )}

            {saveMutation.error && (
              <div className="flex items-center gap-2 text-red-400">
                <AlertCircle className="h-4 w-4" />
                <span className={isMobile ? "text-sm" : "text-xs"}>
                  {saveMutation.error.message || "Failed to save credentials"}
                </span>
              </div>
            )}

            {testResult && (
              <div className={`flex items-center gap-2 ${
                testResult.success ? "text-emerald-400" : "text-red-400"
              }`}>
                {testResult.success ? (
                  <CheckCircle className="h-4 w-4" />
                ) : (
                  <AlertCircle className="h-4 w-4" />
                )}
                <span className={isMobile ? "text-sm" : "text-xs"}>
                  {testResult.message}
                </span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* SSH Key Management */}
      {isSSH && (
        <SettingsTabCredentialsSSH
          isMobile={isMobile}
          repoId={repoId}
          inputClasses={inputClasses}
          buttonHeight={buttonHeight}
          storedSSHKeyPath={originCredential?.ssh_key_path}
          onCredentialsSaved={() => credentialsQuery.refetch()}
        />
      )}
    </div>
  );
}
