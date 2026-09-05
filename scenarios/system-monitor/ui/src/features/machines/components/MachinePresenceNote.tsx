// DOC: docs/reference/cross-platform-effort/machine-linking-ux-2026-08-26.html#screen-07
//
// What the reader has to decide when a remote subject goes quiet is whether to
// do anything. "Vrooli is retrying and will reconnect on its own" is the
// sentence that decides it: without it, a person power-cycles a machine that
// was going to come back by itself.
//
// The note lives in the body rather than in the identity strip so the strip
// keeps one height in every state, and so the explanation sits where a reader
// looks for content.
import { useEffect, useState } from 'react';
import { describeMachine, machinePresenceTone } from '../presence';
import type { Machine } from '../../../types';

interface MachinePresenceNoteProps {
  machine: Machine;
  isStale: boolean;
  lastSuccessfulFetch: Date | null;
  /** Consecutive failed attempts, as counted by the polling loop. */
  retryAttempt: number;
  /** The real polling period for this subject — not a hard-coded promise. */
  retryIntervalSeconds: number;
  /** When the loop last tried, used to count down to the next attempt. */
  lastAttemptAt: Date | null;
}

const clockLabel = (date: Date): string =>
  date.toLocaleTimeString(undefined, { hour12: false });

export const MachinePresenceNote = ({
  machine,
  isStale,
  lastSuccessfulFetch,
  retryAttempt,
  retryIntervalSeconds,
  lastAttemptAt
}: MachinePresenceNoteProps) => {
  const presence = describeMachine(machine);
  const tone = machinePresenceTone(presence, isStale);

  // Only tick while there is a countdown to move.
  const [, setTick] = useState(0);
  useEffect(() => {
    if (tone !== 'stale') return undefined;
    const id = setInterval(() => { setTick(value => value + 1); }, 1000);
    return () => { clearInterval(id); };
  }, [tone]);

  if (tone === 'live' || tone === 'local') {
    return null;
  }

  if (tone === 'unreachable') {
    return (
      <div className="card machine-note" data-tone="unreachable" data-testid="machine-presence-note">
        <h2 className="machine-note__title">Nothing to read from {machine.name}</h2>
        <p>
          {machine.name} is registered with the bridge but cannot be dispatched to, so system-monitor
          has no way to ask it for readings. Linking it again from vrooli-bridge restores the channel.
        </p>
        {/* The age is the fact that separates "it just went down" from "nobody
            has used this machine in a week", and it changes what a reader does
            next. The strip has no room for it; this card does. */}
        <p className="machine-note__meta">
          {presence.platform}
          {presence.age ? ` · last seen ${presence.age} ago` : ' · never seen'}
        </p>
        {machine.readiness && machine.readiness.length > 0 && (
          <ul className="machine-note__readiness">
            {machine.readiness.map(fact => (
              <li key={fact.identity} data-passed={fact.passed}>
                <span aria-hidden="true">{fact.passed ? '✓' : '✕'}</span>
                <span className="sr-only">{fact.passed ? 'passing' : 'failing'}</span>
                {fact.identity.replace(/_/g, ' ')}
              </li>
            ))}
          </ul>
        )}
      </div>
    );
  }

  const secondsToNextRetry = lastAttemptAt
    ? Math.max(0, Math.ceil((lastAttemptAt.getTime() + retryIntervalSeconds * 1000 - Date.now()) / 1000))
    : null;

  return (
    <div className="card machine-note" data-tone="stale" data-testid="machine-presence-note">
      <h2 className="machine-note__title">Every reading below is frozen at one moment</h2>
      <p>
        The channel to {machine.name} closed. Vrooli is retrying every {retryIntervalSeconds} seconds
        and will reconnect on its own when the machine is reachable again.
      </p>
      <p className="machine-note__meta">
        {lastSuccessfulFetch ? `last reading ${clockLabel(lastSuccessfulFetch)}` : 'no reading yet'}
        {secondsToNextRetry !== null ? ` · next retry in ${secondsToNextRetry}s` : ''}
        {retryAttempt > 0 ? ` · attempt ${retryAttempt}` : ''}
      </p>
    </div>
  );
};
