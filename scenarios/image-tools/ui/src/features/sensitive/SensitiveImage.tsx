import { forwardRef, useContext, useEffect, useState, type ImgHTMLAttributes } from "react";
import { Eye, EyeOff } from "lucide-react";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";
import { SettingsContext } from "../settings/settingsContext";
import { DEFAULT_SETTINGS } from "../settings/useSettings";
import { isSensitiveUrl } from "../../lib/sensitiveMedia";

export interface SensitiveImageProps extends ImgHTMLAttributes<HTMLImageElement> {
  src: string;
  /**
   * Whether the image is flagged. Leave unset for in-memory images: the flag is
   * then looked up by object URL (see `markSensitive`).
   */
  sensitive?: boolean;
  /** Classes for the wrapper that appears around a blurred image (layout). */
  wrapperClassName?: string;
  /**
   * False when the image sits inside another control (a button or link): the
   * image stays blurred and the surrounding control's own action applies, since a
   * nested button is invalid. Reveal then happens where the image opens.
   */
  interactive?: boolean;
  /** Icon-only reveal control, for thumbnails too small for a caption. */
  compact?: boolean;
}

/**
 * An `<img>` that blurs when the image is sensitive and the operator's setting is
 * "blur" (the default), with a tap-to-reveal control. Unflagged images render as
 * a bare `<img>`, exactly as before. The reveal is per image instance and resets
 * when the source changes, so a new result is never shown unblurred because the
 * previous one was revealed.
 */
export const SensitiveImage = forwardRef<HTMLImageElement, SensitiveImageProps>(function SensitiveImage(
  { src, sensitive, wrapperClassName, interactive = true, compact = false, className, alt, ...rest },
  ref,
) {
  const { t } = useTranslation();
  const preference = (useContext(SettingsContext)?.settings ?? DEFAULT_SETTINGS).sensitiveImages;
  const flagged = sensitive ?? isSensitiveUrl(src);
  const [revealed, setRevealed] = useState(false);

  useEffect(() => setRevealed(false), [src]);

  if (!flagged || preference === "show") {
    return <img ref={ref} src={src} alt={alt} className={className} {...rest} />;
  }
  const blurred = !revealed;
  return (
    <span data-sensitive={blurred ? "blurred" : "revealed"} className={cn("relative block overflow-hidden", wrapperClassName)}>
      <span className={cn("block", blurred && "scale-110 blur-2xl")}>
        <img ref={ref} src={src} alt={blurred ? "" : alt} aria-hidden={blurred || undefined} className={className} {...rest} />
      </span>
      {interactive ? (
        <button
          type="button"
          aria-pressed={revealed}
          onClick={() => setRevealed((v) => !v)}
          className={cn(
            "flex items-center justify-center gap-1 text-xs font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
            blurred
              ? "absolute inset-0 bg-app-surface/40 text-app-foreground"
              : "absolute bottom-1.5 right-1.5 rounded-control bg-app-surface/90 px-2 py-1 text-app-foreground",
          )}
        >
          {blurred ? <EyeOff aria-hidden="true" className="h-4 w-4" /> : <Eye aria-hidden="true" className="h-4 w-4" />}
          <span className={cn(compact && "sr-only")}>
            {t(blurred ? strings.sensitive.reveal : strings.sensitive.hide)}
          </span>
        </button>
      ) : (
        <span aria-hidden="true" className="absolute inset-0 flex items-center justify-center bg-app-surface/40">
          <EyeOff className="h-4 w-4 text-app-foreground" />
        </span>
      )}
    </span>
  );
});
