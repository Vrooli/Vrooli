import type { Style } from "../../api/studio";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * The candidate behind real copy.
 *
 * The judgement this component exists to support is narrow and specific: *can
 * someone read a headline against this image?* A preview that draws grey bars
 * where the copy goes cannot answer it — an operator judging a backdrop from
 * bars is judging the bars. So the type is real type, at a real size, in the
 * style's own declared text colour, over the reserved region the style declares.
 *
 * Two layouts, because the two destinations fail differently. A landing page
 * puts copy over the image and asks whether it stays legible; a store listing
 * puts a device in the middle and asks whether anything survives around it.
 */
export type MockupKind = "landing" | "store";

export function MockupPreview({
  imageURL,
  style,
  kind,
  failingRegions = [],
}: {
  imageURL?: string;
  style: Style;
  kind: MockupKind;
  /** Indices of reserved regions whose legibility or perceptual check failed. */
  failingRegions?: number[];
}) {
  const { t } = useTranslation();
  const overlay = style.regions.find((region) => region.kind === "overlay");
  const ink = overlay?.textColor || "#ffffff";

  return (
    <figure
      className="flex flex-col gap-2"
      data-testid={`mockup-${kind}`}
      data-mockup-kind={kind}
    >
      <figcaption className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground">
        {kind === "landing"
          ? t(strings.pages.style.mockupLanding)
          : t(strings.pages.style.mockupStore)}
      </figcaption>
      <div className="relative overflow-hidden rounded-panel border border-app-border bg-app-surface-muted">
        {imageURL ? (
          <img
            src={imageURL}
            alt={t(strings.pages.style.mockupAlt, { style: style.name })}
            className="block w-full"
          />
        ) : (
          <div className="aspect-[2/1] w-full" />
        )}

        {kind === "landing" ? (
          <div
            className="absolute flex flex-col gap-3"
            style={{
              left: `${(overlay?.x ?? 0.06) * 100}%`,
              top: `${(overlay?.y ?? 0.26) * 100}%`,
              width: `${(overlay?.width ?? 0.5) * 100}%`,
              color: ink,
            }}
          >
            <p className="text-[0.7vw] font-semibold uppercase tracking-[0.2em] opacity-80">
              {t(strings.pages.style.copyKicker)}
            </p>
            <p className="text-[2.4vw] font-semibold leading-tight">
              {t(strings.pages.style.copyHeadline)}
            </p>
            <p className="text-[1vw] leading-snug opacity-90">
              {t(strings.pages.style.copySubhead)}
            </p>
            <span className="w-fit rounded-control border px-3 py-1 text-[0.9vw] font-medium" style={{ borderColor: ink }}>
              {t(strings.pages.style.copyCta)}
            </span>
          </div>
        ) : (
          // A device outline in the middle, which is what the store chrome
          // actually puts there. The backdrop is judged on what is left.
          <div className="absolute inset-0 flex items-center justify-center">
            <div
              className="h-[68%] w-[34%] rounded-[6%] border-4 opacity-90"
              style={{ borderColor: ink }}
              aria-hidden="true"
            />
          </div>
        )}

        {style.regions.map((region, index) =>
          failingRegions.includes(index) ? (
            <div
              key={`${region.x}-${region.y}-${index}`}
              className="absolute border-2 border-dashed"
              style={{
                left: `${region.x * 100}%`,
                top: `${region.y * 100}%`,
                width: `${region.width * 100}%`,
                height: `${region.height * 100}%`,
                borderColor: "#ff4d4f",
              }}
              data-testid={`failing-region-${index}`}
              role="img"
              aria-label={t(strings.pages.style.regionFailing, { index: index + 1 })}
            />
          ) : null,
        )}
      </div>
    </figure>
  );
}
