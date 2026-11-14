/**
 * The desk's mark: a lens over three lines of research, the desk's role
 * (it watches the work; it is never in the loop). Single colour, reads at 24 px.
 */
export function BrandMark() {
  return (
    <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3 6h8M3 11h5M3 16h4" />
      <circle cx="15.5" cy="12.5" r="4.5" />
      <path d="M18.8 15.8 21 18" />
    </svg>
  );
}
