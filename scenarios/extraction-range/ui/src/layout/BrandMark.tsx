/**
 * The scenario's mark, shown beside the name in the sidebar and alone in the
 * rail. This is scenario-owned on purpose: it is the one piece of chrome that
 * should differ between products. Replace the glyph when real branding exists;
 * keep it a single-colour SVG that reads at 24 px.
 */
export function BrandMark() {
  return (
    <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M4 7h10M4 12h16M4 17h7" />
      <circle cx="17" cy="7" r="1.8" fill="currentColor" stroke="none" />
      <circle cx="14" cy="17" r="1.8" fill="currentColor" stroke="none" />
    </svg>
  );
}
