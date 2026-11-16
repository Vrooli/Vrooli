import { useEffect, useMemo, useRef, useState } from "react";
import type { AskAttachment } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Bot, ShieldCheck } from "lucide-react";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { useTheme } from "../../../theme/themeContext";
import { BRIDGE_PROTOCOL, MIN_FRAME_HEIGHT, parseBridgeMessage } from "./bridge";
import frameBridgeJs from "./frameBridge.js?raw";
import { buildExplainerDocument } from "./document";
import kitCss from "./kit/kit.css?raw";
import kitJs from "./kit/kit.js?raw";
import { pageColorScheme, pageTheme } from "./theme";
import { useExplainerPage } from "./useExplainerPage";

interface ExplainerFrameProps {
  askId: string;
  explainer: AskAttachment;
  attachments: readonly AskAttachment[];
  /** The page asked to open an attachment (already known to be a valid name). */
  onOpen: (name: string, path: string) => void;
  /** The page asked to fill one question of the Hub form; the operator still sends. */
  onPrefill: (questionId: string, value: unknown) => void;
}

/**
 * The agent's own HTML page, sealed. It is untrusted code, so it runs in an
 * `allow-scripts` sandbox without `allow-same-origin` (opaque origin: none of
 * the Hub's cookies, storage or API), under a CSP that allows no network. It
 * can inform and pre-fill; only the Hub's own form answers.
 */
export function ExplainerFrame({ askId, explainer, attachments, onOpen, onPrefill }: ExplainerFrameProps) {
  const { t } = useTranslation();
  const { resolved: theme } = useTheme();
  const page = useExplainerPage(askId, explainer, attachments);
  const frame = useRef<HTMLIFrameElement>(null);
  const [height, setHeight] = useState(MIN_FRAME_HEIGHT * 2);
  // The theme is part of the document only when it is built. A later switch is
  // posted to the running page, so toggling dark mode never reloads it.
  const initialTheme = useRef(theme);
  const srcDoc = useMemo(
    () =>
      page.data
        ? buildExplainerDocument(page.data.html, {
            kitCss,
            kitJs,
            bridgeJs: frameBridgeJs,
            theme: initialTheme.current,
            dataUrls: page.data.dataUrls,
            deferred: page.data.deferred,
          })
        : "",
    [page.data],
  );

  // The frame's own color-scheme is the theme the page is shown in, so the
  // page's prefers-color-scheme and the canvas behind it agree with its colours.
  const scheme = useMemo(() => (page.data ? pageColorScheme(page.data.html, kitCss) : "both"), [page.data]);
  const frameTheme = pageTheme(scheme, theme);

  const known = useRef(attachments);
  useEffect(() => {
    known.current = attachments;
  }, [attachments]);

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      const message = parseBridgeMessage(event, frame.current?.contentWindow ?? null);
      if (!message) return;
      if (message.type === "height") setHeight(message.height);
      else if (message.type === "open") {
        if (known.current.some((attachment) => attachment.name === message.name)) onOpen(message.name, message.path);
      } else onPrefill(message.questionId, message.value);
    };
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [onOpen, onPrefill]);

  useEffect(() => {
    frame.current?.contentWindow?.postMessage({ hub: BRIDGE_PROTOCOL, type: "theme", theme }, "*");
  }, [theme]);

  return (
    // On a phone the page runs the full width of the screen, so its own text sits
    // at the same gutter as the question above it; a panel with a frame, padding
    // and a header stacked the gutters until the text started a tenth of the way in.
    <section data-testid={selectors.packet.explainer} aria-label={t(strings.packet.explainer.label)} className="-mx-[var(--page-gutter)] overflow-hidden border-y border-dashed border-app-warning/60 bg-app-surface md:mx-0 md:rounded-panel md:border">
      <header className="flex flex-wrap items-center gap-x-3 gap-y-0.5 bg-app-surface-muted px-[var(--page-gutter)] py-1.5 text-xs text-app-muted-foreground md:border-b md:border-dashed md:border-app-warning/60 md:px-3">
        <span className="inline-flex items-center gap-1.5 font-semibold uppercase tracking-wide text-app-foreground">
          <Bot size={14} aria-hidden="true" />
          {t(strings.packet.explainer.label)}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <ShieldCheck size={14} aria-hidden="true" />
          {t(strings.packet.explainer.sealed)}
        </span>
      </header>
      {page.isPending && <p role="status" className="px-[var(--page-gutter)] py-6 text-sm text-app-muted-foreground md:px-3">{t(strings.packet.explainer.loading)}</p>}
      {page.isError && (
        <div role="alert" className="flex flex-col items-start gap-3 px-[var(--page-gutter)] py-4 text-sm md:px-3">
          <p>{t(strings.packet.explainer.failed)}</p>
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" onClick={() => void page.refetch()}>
              {t(strings.packet.explainer.retry)}
            </Button>
            <Button type="button" variant="outline" onClick={() => onOpen(explainer.name, "")}>
              {t(strings.packet.explainer.openFile)}
            </Button>
          </div>
        </div>
      )}
      {srcDoc && (
        <iframe
          ref={frame}
          title={t(strings.packet.explainer.frameTitle, { name: explainer.label || explainer.name })}
          sandbox="allow-scripts"
          // The kit's Copy button for a command the operator must run; without
          // it the kit selects the text so a long-press copies it.
          allow="clipboard-write"
          referrerPolicy="no-referrer"
          srcDoc={srcDoc}
          style={{ height, colorScheme: frameTheme }}
          data-page-scheme={scheme}
          className="block w-full border-0 bg-transparent"
        />
      )}
    </section>
  );
}
