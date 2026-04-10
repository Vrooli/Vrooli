import { useState } from "react";
import { GitBranch, GitCommit, Tag, Clock, Copy, Check, AlertTriangle } from "lucide-react";
import type { BuildProvenance } from "../../lib/api";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { writeToClipboard } from "../../lib/browser";

function formatProvenanceTimestamp(iso: string): string {
  const date = new Date(iso);
  if (isNaN(date.getTime())) return iso;
  const now = new Date();
  const isToday = date.toDateString() === now.toDateString();
  if (isToday) {
    return date.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  }
  return (
    date.toLocaleDateString(undefined, { month: "short", day: "numeric" }) +
    " " +
    date.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" })
  );
}

function shortHash(hash: string): string {
  return hash.slice(0, 7);
}

interface ProvenanceCardProps {
  provenance: BuildProvenance;
  compact?: boolean;
}

export function ProvenanceCard({ provenance, compact }: ProvenanceCardProps) {
  const [copied, setCopied] = useState(false);

  const handleCopyHash = async () => {
    const result = await writeToClipboard(provenance.git_commit_hash);
    if (result.success) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  if (compact) {
    return (
      <div className="space-y-1.5">
        {provenance.version && (
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <Tag className="h-3 w-3" />
            <span>v{provenance.version}</span>
          </div>
        )}
        {provenance.git_branch && (
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <GitBranch className="h-3 w-3" />
            <span className="truncate">{provenance.git_branch}</span>
          </div>
        )}
        {provenance.git_commit_hash && (
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <GitCommit className="h-3 w-3" />
            <code
              className="font-mono cursor-help"
              title={provenance.git_commit_hash}
            >
              {shortHash(provenance.git_commit_hash)}
            </code>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => void handleCopyHash()}
              className="h-4 w-4 p-0 text-slate-500 hover:text-slate-300"
              title="Copy commit hash"
            >
              {copied ? (
                <Check className="h-2.5 w-2.5 text-green-400" />
              ) : (
                <Copy className="h-2.5 w-2.5" />
              )}
            </Button>
            {provenance.git_dirty && (
              <Badge variant="outline" className="text-[10px] px-1 py-0 border-yellow-700 text-yellow-400">
                dirty
              </Badge>
            )}
          </div>
        )}
        {provenance.built_at && (
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <Clock className="h-3 w-3" />
            <span>Built {formatProvenanceTimestamp(provenance.built_at)}</span>
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-3 space-y-2">
      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-0.5">
          <p className="text-xs text-slate-500">Version</p>
          <p className="text-sm text-slate-200">{provenance.version || "—"}</p>
        </div>
        <div className="space-y-0.5">
          <p className="text-xs text-slate-500">Branch</p>
          <p className="text-sm text-slate-200 truncate">{provenance.git_branch || "—"}</p>
        </div>
      </div>
      {provenance.git_commit_hash && (
        <div className="space-y-0.5">
          <p className="text-xs text-slate-500">Commit</p>
          <div className="flex items-center gap-2">
            <GitCommit className="h-3.5 w-3.5 text-slate-400" />
            <code
              className="text-sm text-slate-200 font-mono cursor-help"
              title={provenance.git_commit_hash}
            >
              {shortHash(provenance.git_commit_hash)}
            </code>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => void handleCopyHash()}
              className="h-5 w-5 p-0 text-slate-400 hover:text-slate-200"
              title="Copy commit hash"
            >
              {copied ? (
                <Check className="h-3 w-3 text-green-400" />
              ) : (
                <Copy className="h-3 w-3" />
              )}
            </Button>
          </div>
        </div>
      )}
      {provenance.git_dirty && (
        <div className="flex items-center gap-1.5 text-xs text-yellow-400">
          <AlertTriangle className="h-3 w-3" />
          <span>Built with uncommitted changes</span>
        </div>
      )}
      {provenance.built_at && (
        <div className="space-y-0.5">
          <p className="text-xs text-slate-500">Built at</p>
          <p className="text-sm text-slate-200">{formatProvenanceTimestamp(provenance.built_at)}</p>
        </div>
      )}
    </div>
  );
}
