import { useState } from "react";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

type Props = {
  src: string;
  title: string;
  className?: string;
};

function safeSource(value: string): boolean {
  try {
    const url = new URL(value, window.location.href);
    return (url.protocol === "https:" || url.protocol === "http:") && !url.username && !url.password;
  } catch {
    return false;
  }
}

/**
 * Isolates a scenario surface from the Portal workspace. A failed frame is a
 * local unavailable state; it never throws into the chat or shell tree.
 * Sandboxing deliberately grants no native or top-level navigation access.
 */
export function EmbeddedScenarioFrame({ src, title, className }: Props) {
  const { t } = useTranslation();
  const [attempt, setAttempt] = useState(0);
  const [failed, setFailed] = useState(!safeSource(src));
  if (failed) {
    return (
      <div role="alert" className={className} data-testid="embedded-scenario-unavailable">
        <p>{t(strings.companion.embeddedUnavailable)}</p>
        <button type="button" onClick={() => { setFailed(false); setAttempt(value => value + 1); }}>
          {t(strings.companion.retryEmbedded)}
        </button>
      </div>
    );
  }
  const handleFrameError = () => setFailed(true);
  return (
    <iframe
      key={attempt}
      src={src}
      title={title}
      className={className}
      sandbox="allow-forms allow-scripts allow-same-origin"
      referrerPolicy="no-referrer"
      // Resource errors do not bubble consistently across browser engines.
      // Keep an element-level listener alongside React's delegated handler.
      ref={(frame) => { if (frame) frame.onerror = handleFrameError; }}
      onError={handleFrameError}
    />
  );
}
