import { useState, useMemo } from "react";
import {
  RefreshCw,
  Loader2,
  AlertCircle,
  Plus,
  Eye,
  EyeOff,
  Pencil,
  Trash2,
  Key,
  Copy,
  CheckCircle2,
  X,
  AlertTriangle,
  XCircle,
  Info,
} from "lucide-react";
import { useVPSSecrets } from "../../../hooks/useVPSSecrets";
import { useExpectedSecrets, type ExpectedSecret } from "../../../hooks/useExpectedSecrets";
import { cn } from "../../../lib/utils";
import { Button } from "../../ui/button";
import { Input } from "../../ui/input";
import { Alert } from "../../ui/alert";

interface SecretsTabProps {
  deploymentId: string;
}

export function SecretsTab({ deploymentId }: SecretsTabProps) {
  const {
    secrets,
    metadata,
    isLoading,
    error,
    refetch,
    revealSecret,
    hideSecret,
    getSecretValue,
    isRevealed,
    create,
    update,
    delete: deleteSecret,
    isCreating,
    isUpdating,
    isDeleting,
  } = useVPSSecrets(deploymentId);

  const {
    expectedSecrets,
    summary,
    isLoading: isLoadingExpected,
    error: expectedError,
    refetch: refetchExpected,
  } = useExpectedSecrets(deploymentId);

  // Build a set of configured secret keys for quick lookup
  const configuredKeys = useMemo(
    () => new Set(secrets.map((s) => s.key)),
    [secrets]
  );

  // Compute expected secrets with their configuration status
  const expectedSecretsWithStatus = useMemo(() => {
    return expectedSecrets.map((es) => ({
      ...es,
      configured: configuredKeys.has(es.secret_id),
    }));
  }, [expectedSecrets, configuredKeys]);

  // Find additional secrets (on VPS but not in expected list)
  const expectedKeys = useMemo(
    () => new Set(expectedSecrets.map((es) => es.secret_id)),
    [expectedSecrets]
  );
  const additionalSecrets = useMemo(
    () => secrets.filter((s) => !expectedKeys.has(s.key)),
    [secrets, expectedKeys]
  );

  const [showAddModal, setShowAddModal] = useState(false);
  const [prefillKey, setPrefillKey] = useState<string | null>(null);
  const [showEditModal, setShowEditModal] = useState<string | null>(null);
  const [showDeleteModal, setShowDeleteModal] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [revealingKey, setRevealingKey] = useState<string | null>(null);

  // Form state
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [editValue, setEditValue] = useState("");
  const [deleteConfirmation, setDeleteConfirmation] = useState("");
  const [restartScenario, setRestartScenario] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const resetFormState = () => {
    setNewKey("");
    setNewValue("");
    setEditValue("");
    setDeleteConfirmation("");
    setRestartScenario(false);
    setFormError(null);
    setPrefillKey(null);
  };

  const handleCopy = async (key: string) => {
    try {
      const { value, masked } = getSecretValue(key);
      if (masked) {
        // Need to reveal first
        const revealedValue = await revealSecret(key);
        if (revealedValue) {
          await navigator.clipboard.writeText(revealedValue);
          setCopiedKey(key);
          setTimeout(() => setCopiedKey(null), 2000);
        }
      } else {
        await navigator.clipboard.writeText(value);
        setCopiedKey(key);
        setTimeout(() => setCopiedKey(null), 2000);
      }
    } catch {
      setFormError("Failed to copy to clipboard");
    }
  };

  const handleReveal = async (key: string) => {
    if (isRevealed(key)) {
      hideSecret(key);
    } else {
      setRevealingKey(key);
      try {
        await revealSecret(key);
      } catch {
        setFormError("Failed to reveal secret");
      } finally {
        setRevealingKey(null);
      }
    }
  };

  const handleCreate = async () => {
    setFormError(null);
    if (!newKey.trim()) {
      setFormError("Key is required");
      return;
    }
    if (!/^[A-Z][A-Z0-9_]*$/.test(newKey)) {
      setFormError("Key must be uppercase letters, numbers, and underscores");
      return;
    }
    if (!newValue.trim()) {
      setFormError("Value is required");
      return;
    }

    try {
      await create({ key: newKey, value: newValue, restartScenario });
      setShowAddModal(false);
      resetFormState();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to create secret");
    }
  };

  const handleUpdate = async (key: string) => {
    setFormError(null);
    if (!editValue.trim()) {
      setFormError("Value is required");
      return;
    }

    try {
      await update({ key, value: editValue, restartScenario });
      setShowEditModal(null);
      resetFormState();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to update secret");
    }
  };

  const handleDelete = async (key: string) => {
    setFormError(null);
    if (deleteConfirmation !== "DELETE") {
      setFormError("Type DELETE to confirm");
      return;
    }

    try {
      await deleteSecret({ key, restartScenario });
      setShowDeleteModal(null);
      resetFormState();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to delete secret");
    }
  };

  // Handle adding a new secret with optional pre-filled key
  const openAddModal = (key?: string) => {
    resetFormState();
    if (key) {
      setPrefillKey(key);
      setNewKey(key);
    }
    setShowAddModal(true);
  };

  const handleRefreshAll = () => {
    refetch();
    refetchExpected();
  };

  if (isLoading || isLoadingExpected) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-400">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-5 w-5" />
          <span>Failed to load secrets: {error.message}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium text-white">Deployment Secrets</h3>
          <p className="text-sm text-slate-400">
            {summary ? (
              <>
                {summary.configured}/{summary.total} expected secrets configured
                {summary.missing > 0 && summary.required > 0 && (
                  <span className="text-amber-400"> • {summary.required} required missing</span>
                )}
              </>
            ) : (
              <>
                {secrets.length} secret{secrets.length !== 1 ? "s" : ""} on VPS
              </>
            )}
            {metadata?.last_updated && ` • Last updated ${new Date(metadata.last_updated).toLocaleString()}`}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefreshAll}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Button
            size="sm"
            onClick={() => openAddModal()}
          >
            <Plus className="h-4 w-4 mr-2" />
            Add Secret
          </Button>
        </div>
      </div>

      {/* Expected secrets error */}
      {expectedError && (
        <Alert variant="warning" title="Could not load expected secrets">
          {expectedError.message}. Showing VPS secrets only.
        </Alert>
      )}

      {/* Expected Secrets Section */}
      {expectedSecretsWithStatus.length > 0 && (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <h4 className="text-sm font-medium text-slate-300">Expected Secrets</h4>
            <span className="text-xs text-slate-500">
              from service.json
            </span>
          </div>
          <div className="space-y-2">
            {expectedSecretsWithStatus.map((secret) => (
              <ExpectedSecretRow
                key={secret.secret_id}
                secret={secret}
                onAdd={() => openAddModal(secret.secret_id)}
                onEdit={() => {
                  resetFormState();
                  setShowEditModal(secret.secret_id);
                }}
                isConfigured={secret.configured}
              />
            ))}
          </div>
        </div>
      )}

      {/* Info alert */}
      <Alert variant="info" title="Secrets are stored on the VPS">
        Changes here directly modify <code className="text-blue-300">.vrooli/secrets.json</code> on the deployed server.
        Some secrets may require a scenario restart to take effect.
      </Alert>

      {/* Configured Secrets table */}
      {expectedSecretsWithStatus.length > 0 && (
        <div className="flex items-center gap-2 mt-2">
          <h4 className="text-sm font-medium text-slate-300">
            {additionalSecrets.length > 0 ? "All VPS Secrets" : "Configured Secrets"}
          </h4>
          <span className="text-xs text-slate-500">
            on VPS
          </span>
        </div>
      )}
      {secrets.length === 0 ? (
        <div className="p-8 text-center border border-white/10 rounded-lg bg-slate-900/50">
          <Key className="h-12 w-12 text-slate-500 mx-auto mb-3" />
          <h4 className="font-medium text-slate-200">No secrets configured</h4>
          <p className="text-sm text-slate-400 mt-1">
            {expectedSecretsWithStatus.length > 0
              ? "Use the 'Add' buttons above to configure expected secrets"
              : "Click 'Add Secret' to create your first secret"}
          </p>
        </div>
      ) : (
        <div className="border border-white/10 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-800/50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                  Key
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                  Value
                </th>
                <th className="px-4 py-3 text-right text-xs font-medium text-slate-400 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {secrets.map((secret) => {
                const { value, masked } = getSecretValue(secret.key);
                const revealed = isRevealed(secret.key);

                return (
                  <tr key={secret.key} className="hover:bg-white/5">
                    <td className="px-4 py-3">
                      <code className="text-sm font-mono text-slate-200">{secret.key}</code>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <code className={cn(
                          "text-sm font-mono",
                          masked ? "text-slate-500" : "text-emerald-400"
                        )}>
                          {value}
                        </code>
                        {revealed && (
                          <span className="text-xs px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-400">
                            Revealed (30s)
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-1">
                        {/* Reveal/Hide button */}
                        <button
                          onClick={() => handleReveal(secret.key)}
                          disabled={revealingKey === secret.key}
                          className="p-2 text-slate-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                          title={revealed ? "Hide value" : "Reveal value"}
                        >
                          {revealingKey === secret.key ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                          ) : revealed ? (
                            <EyeOff className="h-4 w-4" />
                          ) : (
                            <Eye className="h-4 w-4" />
                          )}
                        </button>

                        {/* Copy button */}
                        <button
                          onClick={() => handleCopy(secret.key)}
                          className="p-2 text-slate-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                          title="Copy value"
                        >
                          {copiedKey === secret.key ? (
                            <CheckCircle2 className="h-4 w-4 text-emerald-400" />
                          ) : (
                            <Copy className="h-4 w-4" />
                          )}
                        </button>

                        {/* Edit button */}
                        <button
                          onClick={() => {
                            resetFormState();
                            setShowEditModal(secret.key);
                          }}
                          className="p-2 text-slate-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                          title="Edit secret"
                        >
                          <Pencil className="h-4 w-4" />
                        </button>

                        {/* Delete button */}
                        <button
                          onClick={() => {
                            resetFormState();
                            setShowDeleteModal(secret.key);
                          }}
                          className="p-2 text-slate-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
                          title="Delete secret"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Add Modal */}
      {showAddModal && (() => {
        // Find expected secret info if adding from expected secrets
        const expectedInfo = prefillKey
          ? expectedSecrets.find((es) => es.secret_id === prefillKey)
          : null;

        return (
          <Modal
            title={prefillKey ? `Add ${prefillKey}` : "Add Secret"}
            onClose={() => {
              setShowAddModal(false);
              setPrefillKey(null);
              resetFormState();
            }}
          >
            <div className="space-y-4">
              {formError && (
                <Alert variant="error" title="Error">
                  {formError}
                </Alert>
              )}

              {/* Show expected secret info when pre-filling */}
              {expectedInfo && (
                <div className="p-3 rounded-lg border border-blue-500/30 bg-blue-500/10">
                  <div className="flex items-start gap-2">
                    <Info className="h-4 w-4 text-blue-400 mt-0.5 flex-shrink-0" />
                    <div className="text-sm">
                      {expectedInfo.label && (
                        <p className="font-medium text-blue-300">{expectedInfo.label}</p>
                      )}
                      {expectedInfo.description && (
                        <p className="text-blue-200/80 mt-0.5">{expectedInfo.description}</p>
                      )}
                      {expectedInfo.default_hint && (
                        <p className="text-blue-200/60 mt-1 text-xs">{expectedInfo.default_hint}</p>
                      )}
                    </div>
                  </div>
                </div>
              )}

              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-300">Key</label>
                <Input
                  value={newKey}
                  onChange={(e) => setNewKey(e.target.value.toUpperCase())}
                  placeholder="MY_API_KEY"
                  className="font-mono uppercase"
                  disabled={!!prefillKey}
                />
                {!prefillKey && (
                  <p className="text-xs text-slate-500">
                    Uppercase letters, numbers, and underscores only
                  </p>
                )}
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-300">Value</label>
                <Input
                  type="password"
                  value={newValue}
                  onChange={(e) => setNewValue(e.target.value)}
                  placeholder="Enter secret value..."
                  className="font-mono"
                  autoFocus={!!prefillKey}
                />
              </div>

              <RestartCheckbox
                checked={restartScenario}
                onChange={setRestartScenario}
              />

              <div className="flex justify-end gap-3 pt-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    setShowAddModal(false);
                    setPrefillKey(null);
                    resetFormState();
                  }}
                >
                  Cancel
                </Button>
                <Button onClick={handleCreate} disabled={isCreating}>
                  {isCreating && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                  Add Secret
                </Button>
              </div>
            </div>
          </Modal>
        );
      })()}

      {/* Edit Modal */}
      {showEditModal && (
        <Modal
          title={`Edit ${showEditModal}`}
          onClose={() => {
            setShowEditModal(null);
            resetFormState();
          }}
        >
          <div className="space-y-4">
            {formError && (
              <Alert variant="error" title="Error">
                {formError}
              </Alert>
            )}

            <div className="space-y-2">
              <label className="text-sm font-medium text-slate-300">New Value</label>
              <Input
                type="password"
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                placeholder="Enter new value..."
                className="font-mono"
              />
            </div>

            <RestartCheckbox
              checked={restartScenario}
              onChange={setRestartScenario}
            />

            <div className="flex justify-end gap-3 pt-2">
              <Button
                variant="outline"
                onClick={() => {
                  setShowEditModal(null);
                  resetFormState();
                }}
              >
                Cancel
              </Button>
              <Button onClick={() => handleUpdate(showEditModal)} disabled={isUpdating}>
                {isUpdating && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                Update Secret
              </Button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Modal */}
      {showDeleteModal && (
        <Modal
          title={`Delete ${showDeleteModal}`}
          onClose={() => {
            setShowDeleteModal(null);
            resetFormState();
          }}
        >
          <div className="space-y-4">
            {formError && (
              <Alert variant="error" title="Error">
                {formError}
              </Alert>
            )}

            <Alert variant="error" title="Warning">
              This action cannot be undone. The secret will be permanently removed from the VPS.
            </Alert>

            <div className="space-y-2">
              <label className="text-sm font-medium text-slate-300">
                Type <code className="text-red-400">DELETE</code> to confirm
              </label>
              <Input
                value={deleteConfirmation}
                onChange={(e) => setDeleteConfirmation(e.target.value)}
                placeholder="DELETE"
                className="font-mono"
              />
            </div>

            <RestartCheckbox
              checked={restartScenario}
              onChange={setRestartScenario}
            />

            <div className="flex justify-end gap-3 pt-2">
              <Button
                variant="outline"
                onClick={() => {
                  setShowDeleteModal(null);
                  resetFormState();
                }}
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={() => handleDelete(showDeleteModal)}
                disabled={isDeleting || deleteConfirmation !== "DELETE"}
              >
                {isDeleting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                Delete Secret
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}

// Modal component
function Modal({
  title,
  children,
  onClose,
}: {
  title: string;
  children: React.ReactNode;
  onClose: () => void;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4">
      <div className="w-full max-w-md rounded-lg border border-white/10 bg-slate-900 p-6 shadow-xl">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-white">{title}</h3>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}

// Restart checkbox component
function RestartCheckbox({
  checked,
  onChange,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex items-start gap-3 p-3 rounded-lg border border-white/10 bg-slate-800/50">
      <input
        type="checkbox"
        id="restart-scenario"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-0.5 h-4 w-4 rounded border-white/20 bg-slate-700 text-blue-500 focus:ring-blue-500"
      />
      <label htmlFor="restart-scenario" className="text-sm">
        <span className="font-medium text-slate-200">Restart scenario after saving</span>
        <p className="text-slate-400 mt-0.5">
          Some secrets require a restart to take effect (e.g., database passwords, API keys)
        </p>
      </label>
    </div>
  );
}

// Expected secret row component with status indicator
function ExpectedSecretRow({
  secret,
  onAdd,
  onEdit,
  isConfigured,
}: {
  secret: ExpectedSecret & { configured: boolean };
  onAdd: () => void;
  onEdit: () => void;
  isConfigured: boolean;
}) {
  // Determine status: configured, optional-missing, required-missing
  const getStatus = () => {
    if (isConfigured) {
      return { icon: CheckCircle2, color: "text-emerald-400", bg: "bg-emerald-500/10", border: "border-emerald-500/30" };
    }
    if (!secret.required) {
      return { icon: AlertTriangle, color: "text-amber-400", bg: "bg-amber-500/10", border: "border-amber-500/30" };
    }
    return { icon: XCircle, color: "text-red-400", bg: "bg-red-500/10", border: "border-red-500/30" };
  };

  const status = getStatus();
  const StatusIcon = status.icon;

  // Classification badge colors
  const classificationColors: Record<string, string> = {
    infrastructure: "bg-purple-500/20 text-purple-300",
    service: "bg-blue-500/20 text-blue-300",
    user: "bg-green-500/20 text-green-300",
  };

  return (
    <div className={cn(
      "flex items-start gap-3 p-3 rounded-lg border",
      status.bg,
      status.border
    )}>
      {/* Status icon */}
      <StatusIcon className={cn("h-5 w-5 mt-0.5 flex-shrink-0", status.color)} />

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <code className="text-sm font-mono font-medium text-slate-200">
            {secret.secret_id}
          </code>
          {secret.label && (
            <span className="text-sm text-slate-400">
              ({secret.label})
            </span>
          )}
          {secret.classification && (
            <span className={cn(
              "text-xs px-1.5 py-0.5 rounded",
              classificationColors[secret.classification] || "bg-slate-500/20 text-slate-300"
            )}>
              {secret.classification}
            </span>
          )}
          {!secret.required && (
            <span className="text-xs px-1.5 py-0.5 rounded bg-slate-500/20 text-slate-400">
              optional
            </span>
          )}
        </div>
        {secret.description && (
          <p className="text-sm text-slate-400 mt-1">
            {secret.description}
          </p>
        )}
        {secret.default_hint && !isConfigured && (
          <p className="text-xs text-slate-500 mt-1 flex items-center gap-1">
            <Info className="h-3 w-3" />
            {secret.default_hint}
          </p>
        )}
      </div>

      {/* Action button */}
      <div className="flex-shrink-0">
        {isConfigured ? (
          <Button
            variant="outline"
            size="sm"
            onClick={onEdit}
            className="h-8"
          >
            <Pencil className="h-3.5 w-3.5 mr-1.5" />
            Edit
          </Button>
        ) : (
          <Button
            size="sm"
            onClick={onAdd}
            className="h-8"
          >
            <Plus className="h-3.5 w-3.5 mr-1.5" />
            Add
          </Button>
        )}
      </div>
    </div>
  );
}
