import { useState } from "react";
import {
  TimeMentionProvider,
  type TimeMentionDisplay,
} from "@vrooli/react-component-library/TimeMention/1";
import { MarkdownRenderer } from "./markdown-renderer";

// The caption identifies the specimen frame; assertions target renderer output.
export function MarkdownStory({ args }: { args: { content: string } }) {
  return (
    <figure>
      <figcaption>Markdown sample</figcaption>
      <div data-testid="markdown-story-output">
        <MarkdownRenderer content={args.content} />
      </div>
    </figure>
  );
}

const SENT = Date.UTC(2026, 9, 29, 1, 27);

/** A message body with its send time: prose times become mentions, code stays literal. */
export function MarkdownTimeStory({ args }: { args: { content: string } }) {
  const [display, setDisplay] = useState<TimeMentionDisplay>("annotate");
  return (
    <figure>
      <figcaption>Message sent Oct 28, 9:27 PM New York time</figcaption>
      <div data-testid="markdown-story-output">
        <TimeMentionProvider
          now={SENT + 4 * 60_000}
          timeZone="America/New_York"
          locale="en-US"
          display={display}
          onDisplayChange={setDisplay}
        >
          <MarkdownRenderer content={args.content} timeAnchor={SENT} />
        </TimeMentionProvider>
      </div>
    </figure>
  );
}
