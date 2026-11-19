import type { Template } from "./spec";

/**
 * A compact, CSS-only representation of a template for pickers and cards. The
 * full composition renders in the artboard; this only needs to carry palette,
 * type and shape so the shelf reads as a set of distinct pages.
 */
export function TemplateThumb({ template }: { template: Template }) {
  const { palette } = template;
  return (
    <span
      className="tpl-thumb"
      style={{
        background: palette.bg,
        color: palette.ink,
        borderColor: palette.line,
      }}
      aria-hidden
    >
      <span className="tpl-thumb-kicker" style={{ color: palette.accent }}>
        {template.category}
      </span>
      <span
        className="tpl-thumb-title"
        style={{
          fontFamily: template.display === "serif" ? "var(--wb-serif)" : "var(--wb-sans)",
        }}
      >
        {template.name}
      </span>
      <span className="tpl-thumb-art">
        <span style={{ background: palette.accent }} />
        <span style={{ background: palette.surfaceAlt }} />
        <span style={{ background: palette.secondary }} />
      </span>
    </span>
  );
}
