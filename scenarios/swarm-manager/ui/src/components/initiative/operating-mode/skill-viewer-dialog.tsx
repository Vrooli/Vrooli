/**
 * SkillViewerDialog
 *
 * Read-only viewer for a single prompt skill. Fetches via the existing
 * `/api/v1/prompts/skills/{id}` endpoint (gated by the prompt catalog allowlist,
 * which already includes operating-mode phase skills) so the LLM-facing skill
 * template a user is about to inspect is reachable from the operating-mode UI
 * without leaving the page.
 *
 * Renders skill name, description, metadata chips (groups, usage type, draft),
 * and the rendered markdown body. Loading and error branches expose retry.
 */

import { useState } from "react";
import { Copy, Check, RefreshCw, Loader2, AlertCircle, ExternalLink } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { selectors } from "../../../consts/selectors";
import { promptService } from "../../../services/prompt-service";
import { useSkillUrl } from "../../../services/external-links";
import { renderMarkdown } from "../../../lib/render-markdown";
import { defaultQueryOptions } from "../../../lib";

export interface SkillViewerDialogProps {
  isOpen: boolean;
  onClose: () => void;
  skillId: string;
}

export function SkillViewerDialog({ isOpen, onClose, skillId }: SkillViewerDialogProps) {
  const [copied, setCopied] = useState(false);
  const externalUrl = useSkillUrl(skillId);

  const skillQuery = useQuery({
    queryKey: ["prompts", "skill", skillId],
    queryFn: () => promptService.getSkill(skillId),
    enabled: isOpen && skillId.length > 0,
    ...defaultQueryOptions,
  });

  const skill = skillQuery.data;
  const isLoading = skillQuery.isLoading || skillQuery.isFetching && !skill;
  const error = skillQuery.error;

  const handleCopyId = async () => {
    try {
      await navigator.clipboard.writeText(skillId);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // Clipboard rejected (insecure context, no permission). Silent — the
      // user can still read the ID off the dialog title.
    }
  };

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title={skill?.name ?? skillId}
      maxWidth="max-w-3xl"
      testId={selectors.initiativeDetails.skillViewerDialog}
    >
      <div className="space-y-4">
        <div className="flex flex-wrap items-center gap-2">
          <code className="rounded bg-slate-800/80 px-1.5 py-0.5 text-[11px] font-mono text-slate-300">
            {skillId}
          </code>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCopyId}
            data-testid={selectors.initiativeDetails.skillViewerCopyId}
          >
            {copied ? (
              <>
                <Check className="mr-1.5 h-3.5 w-3.5 text-emerald-400" />
                Copied
              </>
            ) : (
              <>
                <Copy className="mr-1.5 h-3.5 w-3.5" />
                Copy ID
              </>
            )}
          </Button>
          {skill?.draft && (
            <span className="rounded-full border border-amber-500/40 bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-300">
              Draft
            </span>
          )}
          {skill?.usage_type && (
            <span className="rounded-full border border-slate-700/80 bg-slate-900/60 px-2 py-0.5 text-[11px] text-slate-300">
              {skill.usage_type === "direct_runtime" ? "Direct runtime" : "Support reference"}
            </span>
          )}
          {skill?.groups?.map((group) => (
            <span
              key={group}
              className="rounded-full border border-slate-700/80 bg-slate-900/60 px-2 py-0.5 text-[11px] text-slate-300"
            >
              {group}
            </span>
          ))}
        </div>

        {skill?.description && (
          <p className="text-sm text-slate-300">{skill.description}</p>
        )}

        {isLoading && (
          <div className="flex items-center gap-2 rounded-md border border-slate-800 bg-slate-900/40 p-4 text-sm text-slate-400">
            <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
            Loading skill content…
          </div>
        )}

        {Boolean(error) && !isLoading && (
          <div className="flex flex-wrap items-start justify-between gap-2 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            <p className="flex min-w-0 flex-1 items-center gap-2">
              <AlertCircle className="h-4 w-4 shrink-0" />
              {error instanceof Error ? error.message : "Failed to load skill content."}
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => void skillQuery.refetch()}
              data-testid={selectors.initiativeDetails.skillViewerRetry}
            >
              <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
              Retry
            </Button>
          </div>
        )}

        {skill?.current_content && !isLoading && !error && (
          <div
            className="prose prose-invert max-w-none rounded-md border border-slate-800 bg-slate-900/40 p-4"
            dangerouslySetInnerHTML={{ __html: renderMarkdown(skill.current_content) }}
          />
        )}

        {skill && !skill.current_content && !isLoading && !error && (
          <p className="rounded-md border border-slate-800 bg-slate-900/40 p-4 text-sm italic text-slate-500">
            This skill has no content body.
          </p>
        )}

        <div className="flex flex-wrap items-center justify-end gap-2 pt-1">
          {externalUrl && (
            <a
              href={externalUrl}
              target="_blank"
              rel="noreferrer"
              data-testid={selectors.initiativeDetails.skillViewerExternalLink}
              className="inline-flex items-center gap-1.5 rounded border border-slate-700 bg-slate-800/80 px-3 py-1.5 text-xs font-medium text-slate-200 transition-colors hover:border-cyan-500/60 hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50"
            >
              <ExternalLink className="h-3.5 w-3.5" aria-hidden />
              Open in Prompt Manager
            </a>
          )}
          <Button type="button" variant="outline" size="sm" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
