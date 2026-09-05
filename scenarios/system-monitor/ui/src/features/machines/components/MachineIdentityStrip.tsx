// DOC: docs/reference/cross-platform-effort/machine-linking-ux-2026-08-26.html#screen-06
//
// The strip is a safety feature, not decoration: the failure it prevents is
// reading one machine's 94% disk and acting on a different machine. It is not
// a modal, not a dismissible banner and not a toast, because the condition it
// describes is a persistent property of the window.
//
// It stays one line in every state, so its position and height never move as
// the subject's presence changes. What needs explaining goes in the body, in
// MachinePresenceNote, where a reader looks for content rather than chrome.
import { ArrowLeft, RefreshCw } from 'lucide-react';
import { describeMachine, machinePresenceTone } from '../presence';
import type { Machine } from '../../../types';

interface MachineIdentityStripProps {
  machine: Machine;
  /** The polling loop has failed enough consecutive times to distrust the view. */
  isStale: boolean;
  lastSuccessfulFetch: Date | null;
  onRetry: () => void;
  onBackToLocal: () => void;
}

/** "2 minutes ago" — coarse on purpose; the exact stamp is in the note below. */
const agoLabel = (date: Date): string => {
  const seconds = Math.max(0, Math.floor((Date.now() - date.getTime()) / 1000));
  if (seconds < 60) return `${seconds} seconds ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? '' : 's'} ago`;
  const hours = Math.floor(minutes / 60);
  return `${hours} hour${hours === 1 ? '' : 's'} ago`;
};

export const MachineIdentityStrip = ({
  machine,
  isStale,
  lastSuccessfulFetch,
  onRetry,
  onBackToLocal
}: MachineIdentityStripProps) => {
  const presence = describeMachine(machine);
  const tone = machinePresenceTone(presence, isStale);
  const unreachable = tone === 'unreachable';
  const losing = tone === 'stale';

  return (
    <div className="machine-strip" data-tone={tone} role="status" aria-live="polite">
      <div className="machine-strip__bar">
        <div className="machine-strip__identity">
          <span className={`machine-led machine-led--${tone}`} aria-hidden="true" />
          {losing ? (
            <span>
              <b>{machine.name}</b> stopped responding
              {lastSuccessfulFetch ? ` ${agoLabel(lastSuccessfulFetch)}` : ''} &mdash; showing its last reading
            </span>
          ) : unreachable ? (
            <span>
              <b>{machine.name}</b> is {presence.unreachableReason} &mdash; no readings are available from it
            </span>
          ) : (
            <span>
              Viewing <b>{machine.name}</b> &mdash; {presence.platform}, not this computer
            </span>
          )}
        </div>

        <div className="machine-strip__actions">
          {presence.grantLabel && !unreachable ? (
            <span className="machine-chip" title={presence.grantDetail}>{presence.grantLabel}</span>
          ) : null}
          {losing || unreachable ? (
            <button type="button" className="machine-strip__button" onClick={onRetry}>
              <RefreshCw size={13} aria-hidden="true" />
              Retry now
            </button>
          ) : null}
          <button type="button" className="machine-strip__button" data-testid="back-to-local" onClick={onBackToLocal}>
            <ArrowLeft size={13} aria-hidden="true" />
            Back to this computer
          </button>
        </div>
      </div>
    </div>
  );
};
