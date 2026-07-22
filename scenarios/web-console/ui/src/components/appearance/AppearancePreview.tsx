import { useTranslation } from "react-i18next";
import { DEFAULT_THEME_ID, TERMINAL_THEMES } from "../../consts/config";
import { isLightColor, paneColorStyle, parsePaneColor } from "../../lib/paneColor";
import { strings } from "../../consts/strings";

interface AppearancePreviewProps {
  /** Stored pane-color encoding: "transparent" | "#hex" | "#hex|#hex". */
  headerColor: string;
  themeId: string;
  fontSize: number;
  /** Shown in the fake header; falls back to a generic label. */
  sessionName?: string;
  testIdPrefix?: string;
}

/**
 * Live composite preview of a pane's appearance: the fake header renders the
 * chosen color/gradient exactly like TerminalHeader (via paneColorStyle) and
 * the body renders theme colors at the chosen font size, so the user sees the
 * combination they are building rather than three isolated pickers.
 */
export default function AppearancePreview({
  headerColor,
  themeId,
  fontSize,
  sessionName,
  testIdPrefix = "appearance",
}: AppearancePreviewProps) {
  const { t } = useTranslation();
  const theme = TERMINAL_THEMES[themeId] ?? TERMINAL_THEMES[DEFAULT_THEME_ID];
  const { colors, isTransparent } = parsePaneColor(headerColor);
  const headerStyle = paneColorStyle(headerColor, "header");
  // Gradient headers get a light foreground with a shadow for legibility;
  // solid headers pick the contrast-correct side; transparent inherits chrome.
  const headerTextClass = isTransparent
    ? "text-wc-text-secondary"
    : colors.length > 1
      ? "text-white [text-shadow:0_1px_2px_rgba(0,0,0,0.6)]"
      : isLightColor(colors[0])
        ? "text-black/80"
        : "text-white/90";

  return (
    <div
      data-testid={`${testIdPrefix}-preview`}
      className="overflow-hidden rounded-xl border border-wc-default"
    >
      <div
        className={`flex h-9 items-center gap-2 px-3 ${headerTextClass}`}
        style={headerStyle ?? { background: "rgb(var(--wc-surface-input))" }}
      >
        <span className="flex gap-1.5" aria-hidden="true">
          <span className="h-2 w-2 rounded-full bg-current opacity-40" />
          <span className="h-2 w-2 rounded-full bg-current opacity-40" />
        </span>
        <span className="truncate text-xs font-medium">
          {sessionName || t(strings.appearance.previewSessionFallback)}
        </span>
      </div>
      <div
        className="px-3 py-2.5 font-mono leading-snug"
        style={{
          backgroundColor: theme?.colors.background,
          color: theme?.colors.foreground,
          fontSize: `${fontSize}px`,
        }}
      >
        <div className="truncate">$ vrooli scenario start</div>
        <div className="truncate opacity-70">ready in 1.2s</div>
        <div className="truncate">
          $
          <span
            className="ms-1 inline-block h-[1em] w-[0.55em] translate-y-[0.15em] rounded-[1px]"
            style={{ backgroundColor: theme?.colors.cursor }}
            aria-hidden="true"
          />
        </div>
      </div>
    </div>
  );
}
