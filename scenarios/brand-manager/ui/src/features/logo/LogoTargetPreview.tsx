import type { ContainerStyle } from "@vrooli/proto-types/brand-manager/v1/styles/styles_pb";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { brandMarkUrl } from "../../api/candidates";

const PREVIEW_SIZES = [512, 180, 64, 32, 16] as const;

// defaultStyle mirrors constellation-midnight so the preview is meaningful
// before a brand has a container style assigned.
const defaultStyle = {
  cornerRatio: 0.21875,
  backgroundTop: "#15243c",
  backgroundBottom: "#0b1728",
} as const;

interface LogoTargetPreviewProps {
  brandId: string;
  markAssetId: string;
  style?: ContainerStyle;
}

/**
 * LogoTargetPreview renders the brand's picked mark at every declared size on a
 * light and a dark ground, with the maskable safe-zone circle overlaid at 512.
 * The mark is the raw applied SVG, so the operator sees the real vector the
 * pipeline rasterizes.
 */
export function LogoTargetPreview({ brandId, markAssetId, style }: LogoTargetPreviewProps) {
  const { t } = useTranslation();
  if (!markAssetId) {
    return (
      <section
        data-testid={selectors.logo.preview}
        className="rounded-xl border border-white/10 bg-black/20 p-3 text-sm text-slate-400"
      >
        {t(strings.logo.previewNoMark)}
      </section>
    );
  }
  const cornerRatio = style?.cornerRatio || defaultStyle.cornerRatio;
  const top = style?.backgroundTop || defaultStyle.backgroundTop;
  const bottom = style?.backgroundBottom || defaultStyle.backgroundBottom;
  const markUrl = brandMarkUrl(brandId);

  const grounds = [
    { key: "light", label: t(strings.logo.lightGround), background: "#f8fafc" },
    { key: "dark", label: t(strings.logo.darkGround), background: `linear-gradient(${top}, ${bottom})` },
  ] as const;

  return (
    <section
      data-testid={selectors.logo.preview}
      aria-label={t(strings.logo.previewTitle)}
      className="flex flex-col gap-3 rounded-xl border border-white/10 bg-black/20 p-3"
    >
      <div>
        <h3 className="text-sm font-medium text-slate-400">{t(strings.logo.previewTitle)}</h3>
        <p className="text-xs text-slate-500">{t(strings.logo.previewHint)}</p>
      </div>
      {grounds.map((ground) => (
        <div key={ground.key} className="flex flex-col gap-2">
          <span className="text-xs text-slate-500">{ground.label}</span>
          <div className="flex flex-wrap items-end gap-3">
            {PREVIEW_SIZES.map((size) => (
              <div key={size} className="flex flex-col items-center gap-1">
                <div
                  data-testid={selectors.logo.previewTile}
                  className="relative grid place-items-center"
                  style={{
                    width: size,
                    height: size,
                    background: ground.background,
                    borderRadius: Math.round(cornerRatio * size),
                  }}
                >
                  <img
                    src={markUrl}
                    alt=""
                    style={{ width: `${(style?.markScale || 0.86) * 100}%`, height: `${(style?.markScale || 0.86) * 100}%` }}
                    className="object-contain"
                  />
                  {size === 512 && ground.key === "dark" && (
                    <span
                      data-testid={selectors.logo.previewMaskable}
                      aria-label={t(strings.logo.maskableLabel)}
                      className="pointer-events-none absolute rounded-full border border-cyan-300/60"
                      style={{ width: "80%", height: "80%" }}
                    />
                  )}
                </div>
                <span className="text-xs text-slate-500">
                  {t(strings.logo.sizeLabel, { size })}
                </span>
              </div>
            ))}
          </div>
        </div>
      ))}
    </section>
  );
}
