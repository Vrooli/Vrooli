import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import { Input } from "../ui/input";
import { Loader2, Trash2, CheckCircle, XCircle, AlertCircle } from "lucide-react";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface DeleteButtonProps {
  scenarioName: string;
}

export function DeleteButton({ scenarioName }: DeleteButtonProps) {
  const queryClient = useQueryClient();
  const [showConfirm, setShowConfirm] = useState(false);
  const [confirmText, setConfirmText] = useState("");

  const deleteMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(buildUrl(`/desktop/delete/${scenarioName}`), {
        method: 'DELETE'
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to delete desktop app');
      }
      return res.json();
    },
    onSuccess: () => {
      setShowConfirm(false);
      setConfirmText("");
      queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
    }
  });

  const isDeleting = deleteMutation.isPending;
  const isSuccess = deleteMutation.isSuccess;
  const isFailed = deleteMutation.isError;

  if (isSuccess) {
    return (
      <Badge variant="default" className="gap-1">
        <CheckCircle className="h-3 w-3" />
        Deleted
      </Badge>
    );
  }

  if (isFailed) {
    return (
      <div className="flex gap-2">
        <Badge variant="destructive" className="gap-1">
          <XCircle className="h-3 w-3" />
          Failed
        </Badge>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            deleteMutation.reset();
            setShowConfirm(true);
          }}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (isDeleting) {
    return (
      <div className="flex items-center gap-2">
        <Loader2 className="h-4 w-4 animate-spin text-red-400" />
        <span className="text-sm text-slate-400">Deleting...</span>
      </div>
    );
  }

  if (showConfirm) {
    const isConfirmValid = confirmText === scenarioName;

    return (
      <div className="flex flex-col gap-3 p-4 bg-red-950/20 border border-red-800/30 rounded">
        <div className="flex items-start gap-2">
          <AlertCircle className="h-5 w-5 text-red-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1 text-sm text-red-200">
            <p className="font-semibold text-red-300">⚠️ Permanent Deletion</p>
            <p className="text-red-300/90 mt-1">
              This will <strong>permanently delete</strong> all desktop files for <strong>{scenarioName}</strong>.
            </p>
            <p className="text-red-300/80 mt-2 text-xs">
              This includes generated templates, built packages, and all configuration.
            </p>
            <p className="text-red-300/80 mt-2 text-xs font-semibold">
              This action cannot be undone!
            </p>
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <div>
            <label className="text-xs text-red-300 font-medium">
              Type the scenario name "<span className="font-mono font-bold">{scenarioName}</span>" to confirm:
            </label>
            <Input
              type="text"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder={scenarioName}
              className="mt-1 font-mono text-sm"
              autoFocus
            />
          </div>

          <div className="flex gap-2 justify-end">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setShowConfirm(false);
                setConfirmText("");
              }}
              disabled={isDeleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => deleteMutation.mutate()}
              disabled={isDeleting || !isConfirmValid}
            >
              <Trash2 className="h-3 w-3 mr-1" />
              Confirm Delete
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => setShowConfirm(true)}
      disabled={isDeleting}
      className="gap-1 text-red-400 hover:text-red-300 border-red-800/30 hover:border-red-700/50"
    >
      <Trash2 className="h-4 w-4" />
      Delete
    </Button>
  );
}
