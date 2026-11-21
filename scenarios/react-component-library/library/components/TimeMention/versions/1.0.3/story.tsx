import { useState } from "react";
import { TimeMentionProse, TimeMentionProvider, type TimeMentionDisplay } from "./TimeMention";

// A fixed scene so every capture is identical: a message sent at 9:27 PM New
// York time on Oct 28, 2026, read four minutes later.
const SENT = Date.UTC(2026, 9, 29, 1, 27);
const NOW = SENT + 4 * 60_000;

const MESSAGE = [
  "I'll keep the original 08:53:33 UTC trial cutoff and continue the remaining checks. The demand lease expires at 2026-10-29T03:02:00Z.",
  "The first pass started at lease_ts 1793228013 and ran for 7380s against a PT90M budget. Yesterday's baseline (Wednesday, Oct 27) is preserved, and I'll check back in 20 minutes.",
  "The nightly sweep runs at 02:00 CST and the planner sync is at 9:30.",
];

function Message({ anchored }: { anchored: boolean }) {
  // Each specimen owns its display choice. Left to the shared preference, a
  // switch in one specimen would change every other specimen on the page.
  const [display, setDisplay] = useState<TimeMentionDisplay>("annotate");
  return (
    <TimeMentionProvider
      anchor={anchored ? SENT : null}
      now={NOW}
      timeZone="America/New_York"
      locale="en-US"
      display={display}
      onDisplayChange={setDisplay}
    >
      <article
        data-testid="time-mention-message"
        style={{ maxInlineSize: "38rem", font: "var(--text-body)", lineHeight: 1.7 }}
      >
        {MESSAGE.map((paragraph) => (
          <p key={paragraph}>
            <TimeMentionProse text={paragraph} />
          </p>
        ))}
      </article>
    </TimeMentionProvider>
  );
}

/** A chat message: the send time is known, so every frame is available. */
export function InAMessage() {
  return (
    <main data-rcl-story-background>
      <Message anchored />
    </main>
  );
}

/** The same text where no send time exists, such as a file preview. */
export function WithoutSendTime() {
  return (
    <main data-rcl-story-background>
      <Message anchored={false} />
    </main>
  );
}
