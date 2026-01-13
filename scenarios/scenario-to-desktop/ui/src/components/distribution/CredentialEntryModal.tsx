import { useState, useEffect } from "react";
import { createPortal } from "react-dom";
import { Key, X, Info, AlertTriangle } from "lucide-react";
import { Button } from "../ui/button";
import { Label } from "../ui/label";
import type { CheckCredentialsResponse } from "../../lib/api";

interface CredentialEntryModalProps {
  open: boolean;
  credentialStatus: CheckCredentialsResponse | null;
  onSubmit: (credentials: Record<string, string>) => void;
  onCancel: () => void;
  isSubmitting?: boolean;
}

export function CredentialEntryModal({
  open,
  credentialStatus,
  onSubmit,
  onCancel,
  isSubmitting,
}: CredentialEntryModalProps) {
  const [credentials, setCredentials] = useState<Record<string, string>>({});

  // Reset credentials when modal opens
  useEffect(() => {
    if (open && credentialStatus) {
      const initial: Record<string, string> = {};
      Object.values(credentialStatus.targets).forEach((target) => {
        target.missing_credentials.forEach((envVar) => {
          initial[envVar] = "";
        });
      });
      setCredentials(initial);
    }
  }, [open, credentialStatus]);

  if (!open || !credentialStatus) return null;

  // Get all unique missing credentials across all targets
  const missingCredentials = new Set<string>();
  Object.values(credentialStatus.targets).forEach((target) => {
    target.missing_credentials.forEach((cred) => missingCredentials.add(cred));
  });

  const allFilled = [...missingCredentials].every((key) => credentials[key]?.trim());

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(credentials);
  };

  return createPortal(
    <div
      className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget && !isSubmitting) {
          onCancel();
        }
      }}
    >
      <div className="w-full max-w-lg bg-slate-900 rounded-lg border border-slate-700 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <div className="flex items-center gap-2">
            <Key className="h-5 w-5 text-amber-400" />
            <h2 className="text-lg font-semibold text-slate-100">Enter Credentials</h2>
          </div>
          <button
            onClick={onCancel}
            disabled={isSubmitting}
            className="text-slate-400 hover:text-slate-200 disabled:opacity-50"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Warning */}
        <div className="p-4 border-b border-slate-700 bg-amber-950/20">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
            <div className="text-sm">
              <p className="text-amber-200 font-medium">Missing Environment Variables</p>
              <p className="text-slate-400 mt-1">
                The following credentials are not set in your environment. You can enter them below to proceed with the upload.
              </p>
            </div>
          </div>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          {[...missingCredentials].map((envVar) => (
            <div key={envVar} className="space-y-2">
              <Label htmlFor={`cred-${envVar}`} className="flex items-center gap-2">
                <code className="text-amber-300 bg-slate-800 px-1.5 py-0.5 rounded text-xs">
                  {envVar}
                </code>
              </Label>
              <input
                id={`cred-${envVar}`}
                type="password"
                value={credentials[envVar] || ""}
                onChange={(e) =>
                  setCredentials((prev) => ({ ...prev, [envVar]: e.target.value }))
                }
                placeholder="Enter value..."
                autoComplete="off"
                className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm font-mono"
              />
            </div>
          ))}

          {/* Info box */}
          <div className="rounded-lg bg-slate-800/50 border border-slate-700 p-3">
            <div className="flex items-start gap-2">
              <Info className="h-4 w-4 text-blue-400 flex-shrink-0 mt-0.5" />
              <div className="text-xs text-slate-400 space-y-2">
                <p>
                  <span className="text-slate-300">These credentials are used only for this upload</span> and are not saved anywhere.
                </p>
                <p>
                  To avoid entering credentials each time, set them as environment variables before starting the app:
                </p>
                <pre className="mt-2 p-2 bg-slate-900 rounded text-slate-300 overflow-x-auto">
{[...missingCredentials].map((envVar) => `export ${envVar}="your-value"`).join("\n")}
                </pre>
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={onCancel}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={!allFilled || isSubmitting}>
              {isSubmitting ? "Uploading..." : "Continue with Upload"}
            </Button>
          </div>
        </form>
      </div>
    </div>,
    document.body
  );
}
