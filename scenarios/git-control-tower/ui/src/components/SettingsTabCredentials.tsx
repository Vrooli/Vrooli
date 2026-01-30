import { useState, useEffect } from "react";
import {
  CheckCircle,
  AlertCircle,
  Eye,
  EyeOff,
  ExternalLink,
  RefreshCw,
  Lock,
  Key,
  Copy,
  Trash2,
  Plus,
  ChevronDown,
  ChevronUp
} from "lucide-react";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import {
  useCredentials,
  useSaveCredential,
  useTestCredential,
  useUpdateRemoteURL,
  useSSHKeys,
  useGenerateSSHKey,
  useGetSSHPublicKey,
  useTestSSHConnection,
  useDeleteSSHKey
} from "../lib/hooks";
import type { SSHKeyInfo } from "../lib/api";

interface SettingsTabCredentialsProps {
  remoteUrl?: string;
  hasUpstream?: boolean;
  isMobile: boolean;
}

// URL conversion helpers
function convertSSHToHTTPS(sshUrl: string): string {
  const trimmed = sshUrl.trim();

  // Handle git@host:user/repo.git format
  if (trimmed.startsWith("git@")) {
    const rest = trimmed.slice(4); // Remove "git@"
    const parts = rest.split(":");
    if (parts.length === 2) {
      return `https://${parts[0]}/${parts[1]}`;
    }
  }

  // Handle ssh://git@host/user/repo.git format
  if (trimmed.startsWith("ssh://")) {
    let rest = trimmed.slice(6); // Remove "ssh://"
    rest = rest.replace(/^git@/, "");
    return `https://${rest}`;
  }

  return trimmed;
}

function convertHTTPSToSSH(httpsUrl: string): string {
  const trimmed = httpsUrl.trim();

  if (trimmed.startsWith("https://") || trimmed.startsWith("http://")) {
    let rest = trimmed.replace(/^https?:\/\//, "");
    const parts = rest.split("/");
    if (parts.length >= 2) {
      const host = parts[0];
      const path = parts.slice(1).join("/");
      return `git@${host}:${path}`;
    }
  }

  return trimmed;
}

function detectUrlType(url: string): "https" | "ssh" {
  const trimmed = url.trim();
  if (trimmed.startsWith("git@") || trimmed.startsWith("ssh://")) {
    return "ssh";
  }
  return "https";
}

export function SettingsTabCredentials({
  remoteUrl,
  hasUpstream,
  isMobile
}: SettingsTabCredentialsProps) {
  const credentialsQuery = useCredentials();
  const saveMutation = useSaveCredential();
  const testMutation = useTestCredential();
  const updateUrlMutation = useUpdateRemoteURL();

  // SSH hooks
  const sshKeysQuery = useSSHKeys();
  const generateKeyMutation = useGenerateSSHKey();
  const getPublicKeyMutation = useGetSSHPublicKey();
  const testSSHMutation = useTestSSHConnection();
  const deleteKeyMutation = useDeleteSSHKey();

  const [username, setUsername] = useState("");
  const [token, setToken] = useState("");
  const [showToken, setShowToken] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [testResult, setTestResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);

  // SSH key management state
  const [selectedKeyPath, setSelectedKeyPath] = useState<string | null>(null);
  const [showGenerateForm, setShowGenerateForm] = useState(false);
  const [newKeyType, setNewKeyType] = useState<"ed25519" | "rsa">("ed25519");
  const [newKeyFilename, setNewKeyFilename] = useState("");
  const [newKeyComment, setNewKeyComment] = useState("");
  const [generatedPublicKey, setGeneratedPublicKey] = useState<string | null>(null);
  const [copySuccess, setCopySuccess] = useState(false);
  const [sshTestResult, setSSHTestResult] = useState<{
    success: boolean;
    message: string;
    hint?: string;
    githubUser?: string;
  } | null>(null);

  const urlType = remoteUrl ? detectUrlType(remoteUrl) : "https";
  const isSSH = urlType === "ssh";

  // Get the credential for origin remote
  const originCredential = credentialsQuery.data?.credentials?.find(
    (c) => c.remote === "origin"
  );

  // Get SSH keys
  const sshKeys = sshKeysQuery.data?.keys ?? [];

  // Auto-select first key if none selected
  useEffect(() => {
    if (isSSH && sshKeys.length > 0 && !selectedKeyPath) {
      setSelectedKeyPath(sshKeys[0].path);
    }
  }, [isSSH, sshKeys, selectedKeyPath]);

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

  // Clear SSH test result after 10 seconds
  useEffect(() => {
    if (sshTestResult) {
      const timer = setTimeout(() => setSSHTestResult(null), 10000);
      return () => clearTimeout(timer);
    }
  }, [sshTestResult]);

  // Clear copy success after 2 seconds
  useEffect(() => {
    if (copySuccess) {
      const timer = setTimeout(() => setCopySuccess(false), 2000);
      return () => clearTimeout(timer);
    }
  }, [copySuccess]);

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

  // SSH key management handlers
  const handleGenerateKey = async () => {
    setGeneratedPublicKey(null);
    const result = await generateKeyMutation.mutateAsync({
      type: newKeyType,
      filename: newKeyFilename.trim() || undefined,
      comment: newKeyComment.trim() || undefined
    });

    if (result.success && result.public_key) {
      setGeneratedPublicKey(result.public_key);
      setSelectedKeyPath(result.key?.path ?? null);
      setShowGenerateForm(false);
      setNewKeyFilename("");
      setNewKeyComment("");
    }
  };

  const handleCopyPublicKey = async (keyPath: string) => {
    const result = await getPublicKeyMutation.mutateAsync({ key_path: keyPath });
    if (result.success && result.public_key) {
      await navigator.clipboard.writeText(result.public_key);
      setCopySuccess(true);
    }
  };

  const handleTestSSHConnection = async (keyPath: string) => {
    setSSHTestResult(null);
    const result = await testSSHMutation.mutateAsync({ key_path: keyPath });
    setSSHTestResult({
      success: result.success,
      message: result.message || (result.success ? "Connected!" : "Connection failed"),
      hint: result.hint,
      githubUser: result.github_user
    });
  };

  const handleDeleteKey = async (keyPath: string) => {
    if (!confirm("Are you sure you want to delete this SSH key? This cannot be undone.")) {
      return;
    }
    await deleteKeyMutation.mutateAsync({ key_path: keyPath });
    if (selectedKeyPath === keyPath) {
      setSelectedKeyPath(sshKeys.find(k => k.path !== keyPath)?.path ?? null);
    }
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

  const selectedKey = sshKeys.find(k => k.path === selectedKeyPath);

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
          originCredential?.is_configured
            ? "border-emerald-700/50 bg-emerald-950/20"
            : "border-amber-700/50 bg-amber-950/20"
        } ${isMobile ? "px-4 py-3" : "px-3 py-2"}`}>
          {originCredential?.is_configured ? (
            <>
              <CheckCircle className="h-4 w-4 text-emerald-400 shrink-0" />
              <span className={`text-emerald-200 ${isMobile ? "text-sm" : "text-xs"}`}>
                Credentials configured
                {originCredential.token_masked && (
                  <span className="text-emerald-400/70 ml-1">
                    ({originCredential.token_masked})
                  </span>
                )}
              </span>
            </>
          ) : (
            <>
              <AlertCircle className="h-4 w-4 text-amber-400 shrink-0" />
              <span className={`text-amber-200 ${isMobile ? "text-sm" : "text-xs"}`}>
                No credentials configured
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

              {originCredential?.is_configured && (
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
              )}
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
        <div className={isMobile ? "space-y-4" : "space-y-3"}>
          <h3 className={`font-semibold text-slate-200 ${isMobile ? "text-sm" : "text-xs"}`}>
            SSH Authentication
          </h3>

          {/* SSH Keys List */}
          {sshKeysQuery.isLoading ? (
            <div className="flex items-center gap-2 text-slate-400">
              <RefreshCw className="h-4 w-4 animate-spin" />
              <span className={isMobile ? "text-sm" : "text-xs"}>Loading SSH keys...</span>
            </div>
          ) : sshKeys.length === 0 ? (
            <div className={`rounded-lg border border-slate-700 bg-slate-900/40 ${isMobile ? "px-4 py-4" : "px-3 py-3"}`}>
              <div className="flex items-center gap-2 text-slate-400 mb-3">
                <Key className="h-4 w-4" />
                <span className={isMobile ? "text-sm" : "text-xs"}>No SSH keys found in ~/.ssh</span>
              </div>
              <p className={`text-slate-500 ${isMobile ? "text-xs" : "text-[11px]"}`}>
                Generate a new SSH key to authenticate with GitHub.
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {sshKeys.map((key: SSHKeyInfo) => (
                <div
                  key={key.path}
                  className={`rounded-lg border ${
                    selectedKeyPath === key.path
                      ? "border-blue-600 bg-blue-950/20"
                      : "border-slate-700 bg-slate-900/40"
                  } ${isMobile ? "p-3" : "p-2"} cursor-pointer transition-colors`}
                  onClick={() => setSelectedKeyPath(key.path)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2 min-w-0">
                      <input
                        type="radio"
                        name="ssh-key"
                        checked={selectedKeyPath === key.path}
                        onChange={() => setSelectedKeyPath(key.path)}
                        className="shrink-0"
                      />
                      <Key className="h-4 w-4 text-slate-400 shrink-0" />
                      <span className={`text-slate-200 truncate ${isMobile ? "text-sm" : "text-xs"}`}>
                        {key.filename}
                      </span>
                      <Badge variant="info" className="text-[10px] shrink-0">
                        {key.type.toUpperCase()}
                      </Badge>
                    </div>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDeleteKey(key.path);
                      }}
                      disabled={deleteKeyMutation.isPending}
                      className="p-1 text-slate-500 hover:text-red-400 shrink-0"
                      title="Delete key"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                  {selectedKeyPath === key.path && (
                    <div className={`mt-2 pt-2 border-t border-slate-700/50 ${isMobile ? "space-y-2" : "space-y-1"}`}>
                      <div className={`text-slate-400 ${isMobile ? "text-xs" : "text-[11px]"}`}>
                        <span className="text-slate-500">Fingerprint:</span>{" "}
                        <span className="font-mono">{key.fingerprint || "N/A"}</span>
                      </div>
                      {key.comment && (
                        <div className={`text-slate-400 ${isMobile ? "text-xs" : "text-[11px]"}`}>
                          <span className="text-slate-500">Comment:</span> {key.comment}
                        </div>
                      )}
                      {key.created_at && (
                        <div className={`text-slate-400 ${isMobile ? "text-xs" : "text-[11px]"}`}>
                          <span className="text-slate-500">Created:</span>{" "}
                          {new Date(key.created_at).toLocaleDateString()}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Selected Key Actions */}
          {selectedKey && (
            <div className={`flex gap-2 ${isMobile ? "flex-col" : "flex-row flex-wrap"}`}>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleCopyPublicKey(selectedKey.path)}
                disabled={getPublicKeyMutation.isPending || !selectedKey.has_public}
                className={`${buttonHeight} ${isMobile ? "w-full" : ""}`}
              >
                {getPublicKeyMutation.isPending ? (
                  <RefreshCw className="h-3 w-3 animate-spin mr-2" />
                ) : copySuccess ? (
                  <CheckCircle className="h-3 w-3 mr-2 text-emerald-400" />
                ) : (
                  <Copy className="h-3 w-3 mr-2" />
                )}
                {copySuccess ? "Copied!" : "Copy Public Key"}
              </Button>

              <Button
                variant="outline"
                size="sm"
                onClick={() => handleTestSSHConnection(selectedKey.path)}
                disabled={testSSHMutation.isPending}
                className={`${buttonHeight} ${isMobile ? "w-full" : ""}`}
              >
                {testSSHMutation.isPending ? (
                  <RefreshCw className="h-3 w-3 animate-spin mr-2" />
                ) : null}
                Test Connection
              </Button>

              <a
                href="https://github.com/settings/ssh/new"
                target="_blank"
                rel="noopener noreferrer"
                className={`inline-flex items-center justify-center gap-1 text-blue-400 hover:text-blue-300 border border-slate-700 rounded-md ${buttonHeight} px-3 ${isMobile ? "w-full text-sm" : "text-xs"}`}
              >
                <ExternalLink className="h-3 w-3" />
                Add to GitHub
              </a>
            </div>
          )}

          {/* SSH Test Result */}
          {sshTestResult && (
            <div className={`rounded-lg border ${
              sshTestResult.success
                ? "border-emerald-700/50 bg-emerald-950/20"
                : "border-red-700/50 bg-red-950/20"
            } ${isMobile ? "px-4 py-3" : "px-3 py-2"}`}>
              <div className="flex items-center gap-2">
                {sshTestResult.success ? (
                  <CheckCircle className="h-4 w-4 text-emerald-400 shrink-0" />
                ) : (
                  <AlertCircle className="h-4 w-4 text-red-400 shrink-0" />
                )}
                <span className={`${sshTestResult.success ? "text-emerald-200" : "text-red-200"} ${isMobile ? "text-sm" : "text-xs"}`}>
                  {sshTestResult.message}
                </span>
              </div>
              {sshTestResult.githubUser && (
                <div className={`mt-1 text-emerald-300 ${isMobile ? "text-xs" : "text-[11px]"}`}>
                  Authenticated as: <strong>{sshTestResult.githubUser}</strong>
                </div>
              )}
              {sshTestResult.hint && !sshTestResult.success && (
                <div className={`mt-1 text-slate-400 ${isMobile ? "text-xs" : "text-[11px]"}`}>
                  {sshTestResult.hint}
                </div>
              )}
            </div>
          )}

          {/* Generated Public Key Display */}
          {generatedPublicKey && (
            <div className={`rounded-lg border border-emerald-700/50 bg-emerald-950/20 ${isMobile ? "px-4 py-3" : "px-3 py-2"}`}>
              <div className="flex items-center justify-between mb-2">
                <span className={`text-emerald-200 font-medium ${isMobile ? "text-sm" : "text-xs"}`}>
                  New SSH Key Generated!
                </span>
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(generatedPublicKey);
                    setCopySuccess(true);
                  }}
                  className="p-1 text-emerald-400 hover:text-emerald-300"
                  title="Copy public key"
                >
                  {copySuccess ? <CheckCircle className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                </button>
              </div>
              <p className={`text-slate-400 mb-2 ${isMobile ? "text-xs" : "text-[11px]"}`}>
                Copy the entire line below and paste it into GitHub (the key type, key data, and comment are all required):
              </p>
              <div className={`font-mono text-slate-300 bg-slate-900/60 rounded p-2 break-all ${isMobile ? "text-xs" : "text-[10px]"}`}>
                {generatedPublicKey}
              </div>
              <div className={`mt-2 flex items-center gap-2 ${isMobile ? "flex-col" : "flex-row"}`}>
                <a
                  href="https://github.com/settings/ssh/new"
                  target="_blank"
                  rel="noopener noreferrer"
                  className={`inline-flex items-center gap-1 text-blue-400 hover:text-blue-300 ${isMobile ? "text-sm" : "text-xs"}`}
                >
                  <ExternalLink className="h-3 w-3" />
                  Add this key to GitHub
                </a>
                <button
                  onClick={() => setGeneratedPublicKey(null)}
                  className={`text-slate-500 hover:text-slate-300 ${isMobile ? "text-sm" : "text-xs"}`}
                >
                  Dismiss
                </button>
              </div>
            </div>
          )}

          {/* Generate New Key Section */}
          <div className={`rounded-lg border border-slate-700 bg-slate-900/40 ${isMobile ? "px-4 py-3" : "px-3 py-2"}`}>
            <button
              onClick={() => setShowGenerateForm(!showGenerateForm)}
              className="w-full flex items-center justify-between"
            >
              <div className="flex items-center gap-2">
                <Plus className="h-4 w-4 text-slate-400" />
                <span className={`text-slate-300 ${isMobile ? "text-sm" : "text-xs"}`}>
                  Generate New SSH Key
                </span>
              </div>
              {showGenerateForm ? (
                <ChevronUp className="h-4 w-4 text-slate-400" />
              ) : (
                <ChevronDown className="h-4 w-4 text-slate-400" />
              )}
            </button>

            {showGenerateForm && (
              <div className={`mt-3 pt-3 border-t border-slate-700/50 ${isMobile ? "space-y-4" : "space-y-3"}`}>
                <div className={isMobile ? "space-y-2" : "space-y-1"}>
                  <label className={`text-slate-400 ${isMobile ? "text-sm" : "text-[11px]"}`}>
                    Key Type
                  </label>
                  <div className="flex gap-2">
                    <button
                      onClick={() => setNewKeyType("ed25519")}
                      className={`px-3 py-1.5 rounded border ${
                        newKeyType === "ed25519"
                          ? "border-blue-600 bg-blue-950/40 text-blue-300"
                          : "border-slate-700 bg-slate-900/60 text-slate-400"
                      } ${isMobile ? "text-sm" : "text-xs"}`}
                    >
                      Ed25519 (Recommended)
                    </button>
                    <button
                      onClick={() => setNewKeyType("rsa")}
                      className={`px-3 py-1.5 rounded border ${
                        newKeyType === "rsa"
                          ? "border-blue-600 bg-blue-950/40 text-blue-300"
                          : "border-slate-700 bg-slate-900/60 text-slate-400"
                      } ${isMobile ? "text-sm" : "text-xs"}`}
                    >
                      RSA
                    </button>
                  </div>
                </div>

                <div className={isMobile ? "space-y-2" : "space-y-1"}>
                  <label className={`text-slate-400 ${isMobile ? "text-sm" : "text-[11px]"}`}>
                    Filename (optional)
                  </label>
                  <input
                    type="text"
                    value={newKeyFilename}
                    onChange={(e) => setNewKeyFilename(e.target.value)}
                    placeholder={newKeyType === "ed25519" ? "github_ed25519" : "github_rsa"}
                    className={inputClasses}
                  />
                </div>

                <div className={isMobile ? "space-y-2" : "space-y-1"}>
                  <label className={`text-slate-400 ${isMobile ? "text-sm" : "text-[11px]"}`}>
                    Comment (optional)
                  </label>
                  <input
                    type="text"
                    value={newKeyComment}
                    onChange={(e) => setNewKeyComment(e.target.value)}
                    placeholder="your-email@example.com"
                    className={inputClasses}
                  />
                </div>

                <Button
                  variant="default"
                  size="sm"
                  onClick={handleGenerateKey}
                  disabled={generateKeyMutation.isPending}
                  className={`${buttonHeight} ${isMobile ? "w-full" : ""}`}
                >
                  {generateKeyMutation.isPending ? (
                    <RefreshCw className="h-3 w-3 animate-spin mr-2" />
                  ) : (
                    <Key className="h-3 w-3 mr-2" />
                  )}
                  Generate Key
                </Button>

                {generateKeyMutation.error && (
                  <div className="flex items-center gap-2 text-red-400">
                    <AlertCircle className="h-4 w-4" />
                    <span className={isMobile ? "text-sm" : "text-xs"}>
                      {generateKeyMutation.error.message || "Failed to generate key"}
                    </span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
