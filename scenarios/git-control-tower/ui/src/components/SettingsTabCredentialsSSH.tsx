import { useState, useEffect } from "react";
import {
  CheckCircle,
  AlertCircle,
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
import type { SSHKeyInfo } from "../lib/api";
import {
  useSSHKeys,
  useGenerateSSHKey,
  useGetSSHPublicKey,
  useTestSSHConnection,
  useDeleteSSHKey,
  useSaveCredential,
} from "../lib/hooks";

const EMPTY_SSH_KEYS: SSHKeyInfo[] = [];

interface SettingsTabCredentialsSSHProps {
  isMobile: boolean;
  repoId?: string | null;
  inputClasses: string;
  buttonHeight: string;
  storedSSHKeyPath?: string;
  onCredentialsSaved: () => void;
}

export function SettingsTabCredentialsSSH({
  isMobile,
  repoId,
  inputClasses,
  buttonHeight,
  storedSSHKeyPath,
  onCredentialsSaved,
}: SettingsTabCredentialsSSHProps) {
  const sshKeysQuery = useSSHKeys();
  const generateKeyMutation = useGenerateSSHKey();
  const getPublicKeyMutation = useGetSSHPublicKey();
  const testSSHMutation = useTestSSHConnection();
  const deleteKeyMutation = useDeleteSSHKey();
  const saveMutation = useSaveCredential(repoId);

  const sshKeys = sshKeysQuery.data?.keys ?? EMPTY_SSH_KEYS;

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
  const [sshSaveSuccess, setSSHSaveSuccess] = useState(false);

  // Auto-select stored SSH key, or fall back to first key
  useEffect(() => {
    if (storedSSHKeyPath) {
      const storedKey = sshKeys.find(k => k.path === storedSSHKeyPath);
      if (storedKey && !selectedKeyPath) {
        setSelectedKeyPath(storedKey.path);
        return;
      }
    }
    const firstKey = sshKeys[0];
    if (firstKey && !selectedKeyPath) {
      setSelectedKeyPath(firstKey.path);
    }
  }, [sshKeys, selectedKeyPath, storedSSHKeyPath]);

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

  // Clear SSH save success after 3 seconds
  useEffect(() => {
    if (sshSaveSuccess) {
      const timer = setTimeout(() => setSSHSaveSuccess(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [sshSaveSuccess]);

  const handleSaveSSHKey = async () => {
    if (!selectedKeyPath) return;
    const result = await saveMutation.mutateAsync({
      remote: "origin",
      ssh_key_path: selectedKeyPath
    });
    if (result.success) {
      setSSHSaveSuccess(true);
      onCredentialsSaved();
    }
  };

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

  const selectedKey = sshKeys.find(k => k.path === selectedKeyPath);

  return (
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
        <div className={isMobile ? "space-y-3" : "space-y-2"}>
          <div className={`flex gap-2 ${isMobile ? "flex-col" : "flex-row flex-wrap"}`}>
            <Button
              variant="default"
              size="sm"
              onClick={handleSaveSSHKey}
              disabled={saveMutation.isPending}
              className={`${buttonHeight} ${isMobile ? "w-full" : ""}`}
            >
              {saveMutation.isPending ? (
                <RefreshCw className="h-3 w-3 animate-spin mr-2" />
              ) : (
                <Lock className="h-3 w-3 mr-2" />
              )}
              Save SSH Key
            </Button>

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

          {sshSaveSuccess && (
            <div className="flex items-center gap-2 text-emerald-400">
              <CheckCircle className="h-4 w-4" />
              <span className={isMobile ? "text-sm" : "text-xs"}>
                SSH key saved for git operations
              </span>
            </div>
          )}
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
  );
}
