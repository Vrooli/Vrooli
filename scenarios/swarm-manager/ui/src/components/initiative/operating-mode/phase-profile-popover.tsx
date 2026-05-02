/**
 * PhaseProfilePopover
 *
 * Dialog opened from a phase card's profile chip. Explains what an
 * agent-manager profile is, names the literal profile-key bound to this phase,
 * and (in Phase 2) links out to the profile's detail view in agent-manager.
 *
 * Replaces the layout-shifting native <details> disclosure that previously
 * lived inline on the phase card.
 */

import { ExternalLink } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import { Dialog } from "../../ui/dialog";
import { useAgentProfileUrl } from "../../../services/external-links";

interface PhaseProfilePopoverProps {
  profileKey: string;
  isOpen: boolean;
  onClose: () => void;
}

export function PhaseProfilePopover({ profileKey, isOpen, onClose }: PhaseProfilePopoverProps) {
  const externalUrl = useAgentProfileUrl(profileKey);

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title="Agent profile"
      maxWidth="max-w-md"
      testId={selectors.initiativeDetails.phaseProfilePopover}
    >
      <div className="space-y-4 text-sm text-slate-300">
        <p className="leading-relaxed">
          An <span className="font-medium text-slate-100">agent-manager profile</span> defines
          the model, tool access, and runtime budget the agent gets while running this phase.
          Different phases use different profiles so the right work happens in the right
          environment.
        </p>

        <div>
          <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-400">
            This phase runs as
          </p>
          <code className="block rounded border border-slate-700 bg-slate-800/80 px-2 py-1 font-mono text-[12px] text-slate-100">
            {profileKey}
          </code>
        </div>

        {externalUrl && (
          <div className="flex justify-end pt-1">
            <a
              href={externalUrl}
              target="_blank"
              rel="noreferrer"
              data-testid={selectors.initiativeDetails.phaseProfileExternalLink}
              className="inline-flex items-center gap-1.5 rounded border border-slate-700 bg-slate-800/80 px-3 py-1.5 text-xs font-medium text-slate-200 transition-colors hover:border-cyan-500/60 hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50"
            >
              <ExternalLink className="h-3.5 w-3.5" aria-hidden />
              View profile in Agent Manager
            </a>
          </div>
        )}
      </div>
    </Dialog>
  );
}
