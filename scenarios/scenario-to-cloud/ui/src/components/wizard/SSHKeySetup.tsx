import { useState, useEffect, useCallback, useMemo } from "react";
import {
  Key,
  RefreshCw,
  CheckCircle2,
  AlertCircle,
  XCircle,
  Eye,
  EyeOff,
  Plus,
  Copy,
  Loader2,
  ChevronDown,
  HelpCircle,
} from "lucide-react";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Alert } from "../ui/alert";
import { Collapsible, StatusBadge } from "../ui/collapsible";
import { HelpTooltip } from "../ui/tooltip";
import {
  listSSHKeys,
  generateSSHKey,
  testSSHConnection,
  copySSHKey,
  getPublicKey,
  type SSHKeyInfo,
  type SSHConnectionStatus,
} from "../../lib/api";
import { SSH_ERROR_HINTS } from "../../types/ssh";

interface SSHKeySetupProps {
  host: string;
  port?: number;
  user?: string;
  selectedKeyPath: string | null;
  onKeyPathChange: (keyPath: string | null) => void;
  onConnectionStatusChange: (status: SSHConnectionStatus) => void;
}

export function SSHKeySetup({
  host,
  port = 22,
  user = "root",
  selectedKeyPath,
  onKeyPathChange,
  onConnectionStatusChange,
}: SSHKeySetupProps) {
  // Key discovery state
  const [keys, setKeys] = useState<SSHKeyInfo[]>([]);
  const [isLoadingKeys, setIsLoadingKeys] = useState(false);
  const [keysError, setKeysError] = useState<string | null>(null);
  const [showKeyDropdown, setShowKeyDropdown] = useState(false);

  // Key generation state
  const [showGenerator, setShowGenerator] = useState(false);
  const [genKeyName, setGenKeyName] = useState("vrooli-deploy");
  const [genKeyType, setGenKeyType] = useState<"ed25519" | "rsa">("ed25519");
  const [genKeyPassphrase, setGenKeyPassphrase] = useState("");
  const [showPassphrase, setShowPassphrase] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);
  const [genError, setGenError] = useState<string | null>(null);

  // Connection test state
  const [connectionStatus, setConnectionStatus] = useState<SSHConnectionStatus>("untested");
  const [connectionMessage, setConnectionMessage] = useState<string | null>(null);
  const [connectionHint, setConnectionHint] = useState<string | null>(null);
  const [serverInfo, setServerInfo] = useState<string | null>(null);
  const [isTesting, setIsTesting] = useState(false);

  // Copy key state
  const [showCopyFlow, setShowCopyFlow] = useState(false);
  const [copyPassword, setCopyPassword] = useState("");
  const [showCopyPassword, setShowCopyPassword] = useState(false);
  const [isCopying, setIsCopying] = useState(false);
  const [copyResult, setCopyResult] = useState<{ ok: boolean; message: string } | null>(null);

  // Public key display state
  const [publicKeyContent, setPublicKeyContent] = useState<string | null>(null);
  const [showPublicKey, setShowPublicKey] = useState(false);

  // Load keys on mount
  const loadKeys = useCallback(async () => {
    setIsLoadingKeys(true);
    setKeysError(null);
    try {
      const response = await listSSHKeys();
      setKeys(response.keys);
      // Auto-select first key if none selected
      if (!selectedKeyPath && response.keys.length > 0) {
        onKeyPathChange(response.keys[0].path);
      }
    } catch (e) {
      setKeysError(e instanceof Error ? e.message : "Failed to load SSH keys");
    } finally {
      setIsLoadingKeys(false);
    }
  }, [selectedKeyPath, onKeyPathChange]);

  useEffect(() => {
    loadKeys();
  }, [loadKeys]);

  // Update parent when connection status changes
  useEffect(() => {
    onConnectionStatusChange(connectionStatus);
  }, [connectionStatus, onConnectionStatusChange]);

  // Get selected key info
  const selectedKey = useMemo(() => {
    if (!selectedKeyPath) return null;
    return keys.find((k) => k.path === selectedKeyPath) ?? null;
  }, [keys, selectedKeyPath]);

  // Test connection handler
  const handleTestConnection = async () => {
    if (!selectedKeyPath || !host) return;

    setIsTesting(true);
    setConnectionStatus("testing");
    setConnectionMessage(null);
    setConnectionHint(null);
    setServerInfo(null);
    setCopyResult(null);

    try {
      const response = await testSSHConnection({
        host,
        port,
        user,
        key_path: selectedKeyPath,
      });

      setConnectionStatus(response.status as SSHConnectionStatus);
      setConnectionMessage(response.message ?? null);
      setConnectionHint(response.hint ?? null);
      setServerInfo(response.server_info ?? null);

      // Show copy flow if auth failed
      if (response.status === "auth_failed") {
        setShowCopyFlow(true);
      } else {
        setShowCopyFlow(false);
      }
    } catch (e) {
      setConnectionStatus("unknown_error");
      setConnectionMessage(e instanceof Error ? e.message : "Connection test failed");
    } finally {
      setIsTesting(false);
    }
  };

  // Generate key handler
  const handleGenerateKey = async () => {
    if (!genKeyName) return;

    setIsGenerating(true);
    setGenError(null);

    try {
      const response = await generateSSHKey({
        type: genKeyType,
        filename: genKeyName,
        password: genKeyPassphrase || undefined,
        comment: "Generated by Vrooli for VPS deployment",
      });

      // Add new key to list and select it
      setKeys((prev) => [response.key, ...prev]);
      onKeyPathChange(response.key.path);
      setShowGenerator(false);
      setGenKeyName("vrooli-deploy");
      setGenKeyPassphrase("");
    } catch (e) {
      setGenError(e instanceof Error ? e.message : "Failed to generate key");
    } finally {
      setIsGenerating(false);
    }
  };

  // Copy key handler
  const handleCopyKey = async () => {
    if (!selectedKeyPath || !host || !copyPassword) return;

    setIsCopying(true);
    setCopyResult(null);

    try {
      const response = await copySSHKey({
        host,
        port,
        user,
        key_path: selectedKeyPath,
        password: copyPassword,
      });

      setCopyResult({
        ok: response.ok,
        message: response.message ?? (response.ok ? "Key copied successfully" : "Failed to copy key"),
      });

      if (response.ok) {
        setCopyPassword("");
        // Auto-test connection after successful copy
        setTimeout(() => {
          handleTestConnection();
        }, 500);
      }
    } catch (e) {
      setCopyResult({
        ok: false,
        message: e instanceof Error ? e.message : "Failed to copy key",
      });
    } finally {
      setIsCopying(false);
    }
  };

  // Load public key for display
  const handleShowPublicKey = async () => {
    if (!selectedKeyPath) return;

    try {
      const response = await getPublicKey(selectedKeyPath);
      setPublicKeyContent(response.public_key);
      setShowPublicKey(true);
    } catch (e) {
      setPublicKeyContent(null);
    }
  };

  // Copy public key to clipboard
  const handleCopyPublicKey = async () => {
    if (!publicKeyContent) return;
    try {
      await navigator.clipboard.writeText(publicKeyContent);
    } catch {
      // Fallback for older browsers
      const textarea = document.createElement("textarea");
      textarea.value = publicKeyContent;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
    }
  };

  // Get status badge for the collapsible header
  const statusBadge = useMemo(() => {
    if (!selectedKeyPath) {
      return <StatusBadge status="neutral">Not configured</StatusBadge>;
    }
    switch (connectionStatus) {
      case "connected":
        return <StatusBadge status="success">Connected</StatusBadge>;
      case "testing":
        return <StatusBadge status="info">Testing...</StatusBadge>;
      case "auth_failed":
        return <StatusBadge status="warning">Auth failed</StatusBadge>;
      case "host_unreachable":
      case "timeout":
        return <StatusBadge status="error">Unreachable</StatusBadge>;
      case "key_not_found":
        return <StatusBadge status="error">Key missing</StatusBadge>;
      default:
        return <StatusBadge status="neutral">Not tested</StatusBadge>;
    }
  }, [selectedKeyPath, connectionStatus]);

  // Error hints for display
  const errorHints = connectionStatus !== "untested" && connectionStatus !== "testing" && connectionStatus !== "connected"
    ? SSH_ERROR_HINTS[connectionStatus]
    : null;

  return (
    <Collapsible
      title="SSH Configuration"
      badge={statusBadge}
      defaultOpen={true}
    >
      <div className="space-y-6 pt-4">
        {/* Key Selection */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium text-slate-300 flex items-center gap-2">
              <Key className="h-4 w-4" />
              SSH Key
            </label>
            <button
              type="button"
              onClick={loadKeys}
              disabled={isLoadingKeys}
              className="text-xs text-slate-500 hover:text-slate-300 flex items-center gap-1"
            >
              <RefreshCw className={`h-3 w-3 ${isLoadingKeys ? "animate-spin" : ""}`} />
              Refresh
            </button>
          </div>

          {/* Key Dropdown */}
          <div className="relative">
            <button
              type="button"
              onClick={() => setShowKeyDropdown(!showKeyDropdown)}
              className="w-full flex items-center justify-between rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-slate-100 hover:bg-white/5"
            >
              {selectedKey ? (
                <span className="flex items-center gap-2 truncate">
                  <Key className="h-4 w-4 text-slate-500 flex-shrink-0" />
                  <span className="truncate">{selectedKey.path}</span>
                  <span className="text-xs text-slate-500 flex-shrink-0">
                    ({selectedKey.type.toUpperCase()})
                  </span>
                </span>
              ) : isLoadingKeys ? (
                <span className="text-slate-500">Loading keys...</span>
              ) : keys.length === 0 ? (
                <span className="text-slate-500">No keys found</span>
              ) : (
                <span className="text-slate-500">Select a key...</span>
              )}
              <ChevronDown className={`h-4 w-4 text-slate-500 transition-transform ${showKeyDropdown ? "rotate-180" : ""}`} />
            </button>

            {showKeyDropdown && (
              <div className="absolute z-20 mt-1 w-full max-h-48 overflow-auto rounded-lg border border-white/10 bg-slate-900 shadow-lg">
                {keysError ? (
                  <div className="px-3 py-2 text-sm text-red-400">{keysError}</div>
                ) : keys.length === 0 ? (
                  <div className="px-3 py-4 text-center">
                    <p className="text-sm text-slate-500 mb-2">No SSH keys found in ~/.ssh/</p>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setShowKeyDropdown(false);
                        setShowGenerator(true);
                      }}
                    >
                      <Plus className="h-4 w-4 mr-1" />
                      Generate New Key
                    </Button>
                  </div>
                ) : (
                  keys.map((key) => (
                    <button
                      key={key.path}
                      type="button"
                      onClick={() => {
                        onKeyPathChange(key.path);
                        setShowKeyDropdown(false);
                        setConnectionStatus("untested");
                      }}
                      className={`w-full text-left px-3 py-2 hover:bg-slate-800 ${
                        key.path === selectedKeyPath ? "bg-slate-800" : ""
                      }`}
                    >
                      <div className="flex items-center gap-2">
                        <Key className="h-4 w-4 text-slate-500" />
                        <span className="text-sm text-slate-100 truncate flex-1">
                          {key.path.replace(/^.*[/\\]/, "")}
                        </span>
                        <span className="text-xs text-slate-500">
                          {key.type.toUpperCase()}
                        </span>
                        {key.path === selectedKeyPath && (
                          <CheckCircle2 className="h-4 w-4 text-green-400" />
                        )}
                      </div>
                      <div className="text-xs text-slate-500 mt-0.5 truncate ml-6">
                        {key.fingerprint}
                      </div>
                    </button>
                  ))
                )}
              </div>
            )}
          </div>

          {/* Generate New Key button */}
          {keys.length > 0 && !showGenerator && (
            <button
              type="button"
              onClick={() => setShowGenerator(true)}
              className="text-xs text-blue-400 hover:text-blue-300 flex items-center gap-1"
            >
              <Plus className="h-3 w-3" />
              Generate new key
            </button>
          )}
        </div>

        {/* Key Generator */}
        {showGenerator && (
          <div className="p-4 rounded-lg border border-white/10 bg-black/20 space-y-4">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-medium text-slate-300">Generate New SSH Key</h4>
              <button
                type="button"
                onClick={() => setShowGenerator(false)}
                className="text-slate-500 hover:text-slate-300"
              >
                <XCircle className="h-4 w-4" />
              </button>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <Input
                label="Key Name"
                placeholder="vrooli-deploy"
                value={genKeyName}
                onChange={(e) => setGenKeyName(e.target.value)}
                hint={`Will be saved as ~/.ssh/${genKeyName}`}
              />
              <div className="space-y-1.5">
                <label className="block text-sm font-medium text-slate-300">Key Type</label>
                <select
                  value={genKeyType}
                  onChange={(e) => setGenKeyType(e.target.value as "ed25519" | "rsa")}
                  className="w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-slate-100 focus:outline-none focus:ring-2 focus:ring-slate-500"
                >
                  <option value="ed25519">Ed25519 (Recommended)</option>
                  <option value="rsa">RSA 4096</option>
                </select>
                <p className="text-xs text-slate-500">Ed25519 is more secure and faster</p>
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-slate-300">
                Passphrase (optional)
              </label>
              <div className="relative">
                <input
                  type={showPassphrase ? "text" : "password"}
                  value={genKeyPassphrase}
                  onChange={(e) => setGenKeyPassphrase(e.target.value)}
                  placeholder="Leave empty for no passphrase"
                  className="w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 pr-10 text-sm text-slate-100 focus:outline-none focus:ring-2 focus:ring-slate-500"
                />
                <button
                  type="button"
                  onClick={() => setShowPassphrase(!showPassphrase)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300"
                >
                  {showPassphrase ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              <p className="text-xs text-slate-500">
                Passphrase adds security but requires entry on each use
              </p>
            </div>

            {genError && (
              <Alert variant="error" title="Generation Failed">
                {genError}
              </Alert>
            )}

            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setShowGenerator(false)}>
                Cancel
              </Button>
              <Button
                size="sm"
                onClick={handleGenerateKey}
                disabled={!genKeyName || isGenerating}
              >
                {isGenerating ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                    Generating...
                  </>
                ) : (
                  <>
                    <Plus className="h-4 w-4 mr-1" />
                    Generate Key
                  </>
                )}
              </Button>
            </div>
          </div>
        )}

        {/* Connection Test */}
        {selectedKeyPath && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-slate-300">Connection Status</span>
              <Button
                variant="outline"
                size="sm"
                onClick={handleTestConnection}
                disabled={isTesting || !host}
              >
                {isTesting ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                    Testing...
                  </>
                ) : (
                  "Test Connection"
                )}
              </Button>
            </div>

            {/* Status Display */}
            <div
              className={`p-3 rounded-lg border ${
                connectionStatus === "connected"
                  ? "bg-emerald-500/10 border-emerald-500/30"
                  : connectionStatus === "auth_failed"
                    ? "bg-amber-500/10 border-amber-500/30"
                    : connectionStatus === "host_unreachable" || connectionStatus === "timeout"
                      ? "bg-red-500/10 border-red-500/30"
                      : connectionStatus === "testing"
                        ? "bg-blue-500/10 border-blue-500/30"
                        : "bg-slate-800/50 border-slate-700"
              }`}
            >
              <div className="flex items-start gap-2">
                {connectionStatus === "connected" && (
                  <CheckCircle2 className="h-5 w-5 text-emerald-400 flex-shrink-0" />
                )}
                {connectionStatus === "auth_failed" && (
                  <AlertCircle className="h-5 w-5 text-amber-400 flex-shrink-0" />
                )}
                {(connectionStatus === "host_unreachable" || connectionStatus === "timeout" || connectionStatus === "key_not_found") && (
                  <XCircle className="h-5 w-5 text-red-400 flex-shrink-0" />
                )}
                {connectionStatus === "testing" && (
                  <Loader2 className="h-5 w-5 text-blue-400 flex-shrink-0 animate-spin" />
                )}
                {connectionStatus === "untested" && (
                  <HelpCircle className="h-5 w-5 text-slate-500 flex-shrink-0" />
                )}

                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium">
                    {connectionStatus === "connected" && "Connected successfully"}
                    {connectionStatus === "testing" && "Testing connection..."}
                    {connectionStatus === "untested" && "Not tested"}
                    {errorHints && errorHints.title}
                  </p>
                  {serverInfo && (
                    <p className="text-xs text-slate-400 mt-0.5">{serverInfo}</p>
                  )}
                  {connectionMessage && connectionStatus !== "connected" && (
                    <p className="text-xs text-slate-400 mt-0.5">{connectionMessage}</p>
                  )}
                  {errorHints && (
                    <ul className="text-xs text-slate-400 mt-2 space-y-1">
                      {errorHints.hints.map((hint, i) => (
                        <li key={i} className="flex items-start gap-1">
                          <span className="text-slate-600">•</span>
                          {hint}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Copy Key to Server Flow */}
        {showCopyFlow && selectedKeyPath && (
          <div className="p-4 rounded-lg border border-amber-500/30 bg-amber-500/10 space-y-4">
            <div className="flex items-start gap-2">
              <AlertCircle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
              <div>
                <h4 className="text-sm font-medium text-amber-200">Copy SSH Key to Server</h4>
                <p className="text-xs text-amber-300/70 mt-1">
                  Your public key needs to be added to the server's authorized_keys file.
                  Enter the server password to copy it automatically.
                </p>
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="block text-sm font-medium text-slate-300">
                Server Password for {user}@{host}
              </label>
              <div className="relative">
                <input
                  type={showCopyPassword ? "text" : "password"}
                  value={copyPassword}
                  onChange={(e) => setCopyPassword(e.target.value)}
                  placeholder="Enter SSH password"
                  className="w-full rounded-lg border border-white/10 bg-black/30 px-3 py-2 pr-10 text-sm text-slate-100 focus:outline-none focus:ring-2 focus:ring-slate-500"
                />
                <button
                  type="button"
                  onClick={() => setShowCopyPassword(!showCopyPassword)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300"
                >
                  {showCopyPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>

            {copyResult && (
              <Alert variant={copyResult.ok ? "success" : "error"}>
                {copyResult.message}
              </Alert>
            )}

            <div className="flex items-center justify-between">
              <button
                type="button"
                onClick={() => {
                  handleShowPublicKey();
                }}
                className="text-xs text-blue-400 hover:text-blue-300 flex items-center gap-1"
              >
                <Copy className="h-3 w-3" />
                Show public key to copy manually
              </button>
              <Button
                size="sm"
                onClick={handleCopyKey}
                disabled={!copyPassword || isCopying}
              >
                {isCopying ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                    Copying...
                  </>
                ) : (
                  <>
                    <Copy className="h-4 w-4 mr-1" />
                    Copy Key to Server
                  </>
                )}
              </Button>
            </div>

            {/* Public Key Display */}
            {showPublicKey && publicKeyContent && (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs text-slate-400">Public Key:</span>
                  <button
                    type="button"
                    onClick={handleCopyPublicKey}
                    className="text-xs text-blue-400 hover:text-blue-300 flex items-center gap-1"
                  >
                    <Copy className="h-3 w-3" />
                    Copy
                  </button>
                </div>
                <pre className="text-xs text-slate-300 bg-black/30 p-2 rounded border border-white/10 overflow-x-auto whitespace-pre-wrap break-all">
                  {publicKeyContent}
                </pre>
                <p className="text-xs text-slate-500">
                  Add this to <code className="text-slate-400">~/.ssh/authorized_keys</code> on your server
                </p>
              </div>
            )}
          </div>
        )}
      </div>
    </Collapsible>
  );
}
