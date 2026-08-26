/**
 * @libraryId react-component-library:ProgressiveImage
 * @displayName ProgressiveImage
 * @description A responsive image surface that reserves space, reveals content gently, and degrades legibly.
 * @version 1.1.4
 * @tags ["primitive","media","responsive","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

/** @vrooliComponentSource primitives.progressive-image */
import {
  useEffect,
  useState,
  type CSSProperties,
  type ImgHTMLAttributes,
  type ReactNode,
} from "react";
import { AspectRatio } from "../../../AspectRatio/versions/1.0.0/AspectRatio";

const styles = `
[data-rcl-progressive-image] { position: relative; isolation: isolate; color: var(--color-foreground); background: var(--color-surface-muted); border-radius: var(--radius-panel); }
[data-rcl-progressive-image] [data-rcl-progressive-image-placeholder], [data-rcl-progressive-image] [data-rcl-progressive-image-error] { position: absolute; inset: 0; display: grid; place-items: center; padding: var(--space-lg); text-align: center; }
[data-rcl-progressive-image-placeholder] { color: var(--color-muted-foreground); background: linear-gradient(120deg, var(--color-surface-muted), var(--color-surface-raised), var(--color-surface-muted)); background-size: 220% 100%; animation: rcl-progressive-image-shimmer var(--dur-slow) var(--ease-standard) infinite; }
[data-rcl-progressive-image-placeholder][data-visible="false"], [data-rcl-progressive-image-error][data-visible="false"] { opacity: 0; pointer-events: none; }
[data-rcl-progressive-image] img { display: block; inline-size: 100%; block-size: 100%; object-fit: cover; opacity: 0; transform: scale(1.01); transition: opacity var(--dur-slow) var(--ease-standard), transform var(--dur-slow) var(--ease-standard); }
[data-rcl-progressive-image] img[data-loaded="true"] { opacity: 1; transform: scale(1); }
[data-rcl-progressive-image-error] { color: var(--color-danger); background: var(--color-surface-raised); }
[data-rcl-progressive-image-error-content] { display: grid; justify-items: center; gap: var(--space-2xs); max-inline-size: 28rem; }
[data-rcl-progressive-image-error-icon] { display: grid; place-items: center; inline-size: var(--space-xl); block-size: var(--space-xl); border: var(--border-hairline) solid currentColor; border-radius: var(--radius-pill); font: var(--text-label); }
@keyframes rcl-progressive-image-shimmer { 0% { background-position: 100% 0; } 100% { background-position: -100% 0; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-progressive-image-placeholder] { animation: none; } [data-rcl-progressive-image] img { transition: none; transform: none; } }
`;

export interface ProgressiveImageSource {
  srcSet: string;
  media?: string;
  sizes?: string;
  type?: string;
}

export type ProgressiveImageState = "loading" | "loaded" | "error";

export interface ProgressiveImageProps
  extends Omit<ImgHTMLAttributes<HTMLImageElement>, "src" | "alt" | "loading"> {
  src: string;
  alt: string;
  /**
   * Optional externally resolved display state. This is useful when an
   * integration owns image resolution (and for deterministic preview
   * harnesses); when omitted, the browser request owns the state.
   */
  displayState?: ProgressiveImageState;
  ratio?: number | string;
  sources?: ProgressiveImageSource[];
  placeholder?: ReactNode;
  errorFallback?: ReactNode;
  loading?: "eager" | "lazy";
  onLoadingStateChange?: (state: "loading" | "loaded" | "error") => void;
  frameStyle?: CSSProperties;
}

export const ProgressiveImage = withClassName(function ProgressiveImage({
  alt,
  decoding = "async",
  errorFallback,
  frameStyle,
  loading = "lazy",
  onError,
  onLoad,
  onLoadingStateChange,
  placeholder,
  ratio = "16 / 9",
  sources = [],
  src,
  displayState,
  style,
  ...imageProps
}: ProgressiveImageProps) {
  const [observedState, setObservedState] = useState<ProgressiveImageState>("loading");

  useEffect(() => {
    setObservedState("loading");
    onLoadingStateChange?.("loading");
  }, [onLoadingStateChange, src]);

  const updateState = (next: "loaded" | "error") => {
    setObservedState(next);
    onLoadingStateChange?.(next);
  };
  const state = displayState ?? observedState;

  return (
    <>
      <style data-rcl-progressive-image-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <AspectRatio
        data-rcl-progressive-image
        ratio={ratio}
        style={{ ...frameStyle, ...style }}
        aria-busy={state === "loading" || undefined}
      >
        <div
          data-rcl-progressive-image-placeholder
          data-visible={state === "loading"}
          aria-hidden={state !== "loading"}
        >
          {placeholder ?? (
            <span>{translate("primitives.progressive-image.text.1", "Loading image…")}</span>
          )}
        </div>
        <picture>
          {sources.map((source) => (
            <source
              key={`${source.media ?? "all"}-${source.srcSet}-${source.type ?? ""}`}
              {...source}
            />
          ))}
          <img
            {...imageProps}
            src={src}
            alt={alt}
            aria-label={alt}
            decoding={decoding}
            loading={loading}
            data-loaded={state === "loaded"}
            onLoad={(event) => {
              updateState("loaded");
              onLoad?.(event);
            }}
            onError={(event) => {
              updateState("error");
              onError?.(event);
            }}
          />
        </picture>
        <div
          data-rcl-progressive-image-error
          data-visible={state === "error"}
          role="alert"
          aria-live="assertive"
        >
          <div data-rcl-progressive-image-error-content>
            <span data-rcl-progressive-image-error-icon aria-hidden="true">
              !
            </span>
            {errorFallback ?? (
              <span>{translate("primitives.progressive-image.text.2", "Image unavailable.")}</span>
            )}
          </div>
        </div>
      </AspectRatio>
    </>
  );
});
