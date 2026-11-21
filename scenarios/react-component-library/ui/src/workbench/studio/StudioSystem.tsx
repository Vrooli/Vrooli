import { ChevronRight, Frame, Layers, Palette, Shapes } from "lucide-react";

import { useTranslation } from "../../i18n";
import type { DesignFrame } from "../designs";
import type { TemplatePalette, TemplateNodeType } from "../templates/spec";
import { FRAME_SIZE, firstNodeOfType, paletteTokens } from "./model";
import { layerCount, patternUsage, primitiveUsage, type Coverage } from "./rcl";
import { cx } from "../cx";
import { StudioChrome, type ChromeOptions } from "./StudioChrome";
import { StudioExtract } from "./StudioExtract";

const TOKEN_LABEL: Record<keyof TemplatePalette, { key: string; label: string }> = {
  bg: { key: "workbench.studio.token.bg", label: "Background" },
  surface: { key: "workbench.studio.token.surface", label: "Surface" },
  surfaceAlt: { key: "workbench.studio.token.surfaceAlt", label: "Surface alt" },
  ink: { key: "workbench.studio.token.ink", label: "Ink" },
  muted: { key: "workbench.studio.token.muted", label: "Muted ink" },
  line: { key: "workbench.studio.token.line", label: "Line" },
  accent: { key: "workbench.studio.token.accent", label: "Accent" },
  accentInk: { key: "workbench.studio.token.accentInk", label: "Accent ink" },
  accentSoft: { key: "workbench.studio.token.accentSoft", label: "Accent soft" },
  secondary: { key: "workbench.studio.token.secondary", label: "Secondary" },
};

const PATTERN_LABEL: Partial<Record<TemplateNodeType, { key: string; label: string }>> = {
  stack: { key: "workbench.studio.pattern.stack", label: "Stack" },
  row: { key: "workbench.studio.pattern.row", label: "Row" },
  grid: { key: "workbench.studio.pattern.grid", label: "Grid" },
  card: { key: "workbench.studio.pattern.card", label: "Card" },
  surface: { key: "workbench.studio.pattern.surface", label: "Surface" },
  slot: { key: "workbench.studio.pattern.slot", label: "Component slot" },
};

const HEX = /^#[0-9a-f]{6}$/i;

export function StudioSystem({
  theme,
  frames,
  activeFrame,
  token,
  onSelectToken,
  onTokenChange,
  onSelectNode,
  onSelectFrame,
  coverage,
  options,
  onOptionsChange,
}: {
  theme: TemplatePalette;
  frames: DesignFrame[];
  activeFrame: DesignFrame;
  token: keyof TemplatePalette | null;
  onSelectToken: (key: keyof TemplatePalette | null) => void;
  onTokenChange: (key: keyof TemplatePalette, value: string) => void;
  onSelectNode: (id: string) => void;
  onSelectFrame: (id: string) => void;
  coverage: Coverage;
  /** Design-owned chrome; omitted by callers that only read the system. */
  options?: ChromeOptions;
  onOptionsChange?: (patch: ChromeOptions, label: string) => void;
}) {
  const { t } = useTranslation();
  const tokens = paletteTokens(theme);
  const primitives = primitiveUsage(frames, activeFrame.id);
  const patterns = patternUsage(activeFrame.spec);
  const selected = token ? tokens.find((entry) => entry.key === token) : undefined;

  return (
    <>
      <div className="studio-pane-head">
        <span className="studio-pane-title">
          <Palette aria-hidden size={13} />
          {t("workbench.studio.systemTheme", { defaultValue: "Design theme" })}
        </span>
        <span className="studio-frame-chip">
          {t("workbench.studio.usedByFrames", {
            defaultValue: "used by {{count}} frames",
            count: frames.length,
          })}
        </span>
      </div>

      <div className="studio-system">
        {onOptionsChange ? (
          <StudioChrome options={options ?? {}} onChange={onOptionsChange} />
        ) : null}
        <section className="studio-system-group">
          <header className="studio-system-head">
            <span>{t("workbench.studio.tokens", { defaultValue: "Tokens" })}</span>
            <small>
              {t("workbench.studio.tokenCount", {
                defaultValue: "{{count}} · theme",
                count: tokens.length,
              })}
            </small>
          </header>
          {tokens.map((entry) => {
            const label = TOKEN_LABEL[entry.key];
            const current = entry.key === token;
            return (
              <button
                key={entry.key}
                type="button"
                className={cx("studio-token-row", current && "is-selected")}
                aria-pressed={current}
                onClick={() => onSelectToken(current ? null : entry.key)}
              >
                <span className="studio-token-swatch" style={{ background: entry.value }} aria-hidden />
                <span className="studio-token-name">{t(label.key, { defaultValue: label.label })}</span>
                <span className="studio-token-value">{entry.value}</span>
              </button>
            );
          })}

          {selected ? (
            <div className="studio-token-editor">
              <span className="wb-caption">
                {t("workbench.studio.editToken", {
                  defaultValue: "Edit {{token}}",
                  token: t(TOKEN_LABEL[selected.key].key, {
                    defaultValue: TOKEN_LABEL[selected.key].label,
                  }),
                })}
              </span>
              <div className="studio-token-controls">
                {HEX.test(selected.value) ? (
                  <input
                    type="color"
                    className="studio-color"
                    value={selected.value}
                    aria-label={t("workbench.studio.tokenPicker", {
                      defaultValue: "{{token}} colour",
                      token: t(TOKEN_LABEL[selected.key].key, {
                        defaultValue: TOKEN_LABEL[selected.key].label,
                      }),
                    })}
                    onChange={(event) => onTokenChange(selected.key, event.target.value)}
                  />
                ) : null}
                <input
                  className="wb-input studio-token-input"
                  value={selected.value}
                  aria-label={t("workbench.studio.tokenValue", {
                    defaultValue: "{{token}} value",
                    token: t(TOKEN_LABEL[selected.key].key, {
                      defaultValue: TOKEN_LABEL[selected.key].label,
                    }),
                  })}
                  onChange={(event) => onTokenChange(selected.key, event.target.value)}
                />
              </div>
              <p className="studio-note">
                {t("workbench.studio.tokenNote", {
                  defaultValue: "Applied live to every frame. Each edit is one undoable step.",
                })}
              </p>
            </div>
          ) : null}
        </section>

        <div className="studio-system-arrow" aria-hidden>
          <ChevronRight size={12} />
        </div>

        <section className="studio-system-group">
          <header className="studio-system-head">
            <span>{t("workbench.studio.primitives", { defaultValue: "Primitives" })}</span>
            <small>
              {t("workbench.studio.primitiveCount", {
                defaultValue: "{{count}} used",
                count: primitives.length,
              })}
            </small>
          </header>
          {primitives.length === 0 ? (
            <p className="studio-note">
              {t("workbench.studio.noPrimitives", {
                defaultValue: "This frame has no RCL primitives yet.",
              })}
            </p>
          ) : (
            primitives.map((entry) => (
              <button
                key={entry.type}
                type="button"
                className="studio-system-row"
                onClick={() => {
                  const id = firstNodeOfType(activeFrame.spec, entry.type);
                  if (id) onSelectNode(id);
                }}
              >
                <Shapes aria-hidden size={12} className="studio-system-icon" />
                <span className="studio-system-name">{entry.name}</span>
                <span className="studio-system-meta">
                  {t("workbench.studio.uses", {
                    defaultValue: "{{count}} uses · {{frames}} frames",
                    count: entry.count,
                    frames: entry.frames,
                  })}
                </span>
              </button>
            ))
          )}
        </section>

        <div className="studio-system-arrow" aria-hidden>
          <ChevronRight size={12} />
        </div>

        <section className="studio-system-group">
          <header className="studio-system-head">
            <span>{t("workbench.studio.patterns", { defaultValue: "Patterns" })}</span>
            <small>
              {t("workbench.studio.layerCountShort", {
                defaultValue: "{{count}} layers",
                count: layerCount(activeFrame.spec),
              })}
            </small>
          </header>
          {patterns.length === 0 ? (
            <p className="studio-note">
              {t("workbench.studio.noPatterns", {
                defaultValue: "No grouping layers in this frame.",
              })}
            </p>
          ) : (
            patterns.map((entry) => {
              const label = PATTERN_LABEL[entry.type];
              return (
                <div key={entry.type} className="studio-system-row is-static">
                  <Layers aria-hidden size={12} className="studio-system-icon" />
                  <span className="studio-system-name">
                    {label ? t(label.key, { defaultValue: label.label }) : entry.type}
                  </span>
                  <span className="studio-system-meta">
                    {t("workbench.studio.usesShort", {
                      defaultValue: "{{count}}",
                      count: entry.count,
                    })}
                  </span>
                </div>
              );
            })
          )}
        </section>

        <div className="studio-system-arrow" aria-hidden>
          <ChevronRight size={12} />
        </div>

        <section className="studio-system-group">
          <header className="studio-system-head">
            <span>{t("workbench.studio.frames", { defaultValue: "Pages" })}</span>
            <small>
              {t("workbench.studio.frameCount", {
                defaultValue: "{{count}}",
                count: frames.length,
              })}
            </small>
          </header>
          {frames.map((frame) => {
            const current = frame.id === activeFrame.id;
            const size = FRAME_SIZE[frame.device];
            return (
              <button
                key={frame.id}
                type="button"
                className={cx("studio-system-row", current && "is-current")}
                aria-current={current ? "true" : undefined}
                onClick={() => onSelectFrame(frame.id)}
              >
                <Frame aria-hidden size={12} className="studio-system-icon" />
                <span className="studio-system-name">{frame.name}</span>
                <span className="studio-system-meta">
                  {current
                    ? t("workbench.studio.current", { defaultValue: "current" })
                    : `${size.width} × ${size.height}`}
                </span>
              </button>
            );
          })}
        </section>
        <StudioExtract spec={activeFrame.spec} onSelect={onSelectNode} />
      </div>

      <div className="studio-pane-foot">
        <span className="studio-coverage">
          {t("workbench.studio.chainCoverage", {
            defaultValue: "{{percent}}% of layers resolve to this chain",
            percent: Math.round(coverage.ratio * 100),
          })}
        </span>
      </div>
    </>
  );
}
