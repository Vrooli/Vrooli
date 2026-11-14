/**
 * A path or code span the sender wrote as itself: monospace, left to right, and
 * allowed to break anywhere, because it has no spaces to break at and would
 * otherwise overflow a phone.
 */
export function Literal({ children, inHeading = false }: { children: string; /** In a heading it drops to body size and weight, so a long path doesn't shout. */ inHeading?: boolean }) {
  return (
    <code dir="ltr" className={`font-mono [overflow-wrap:anywhere] ${inHeading ? "text-base font-normal" : "text-[0.9em]"}`}>
      {children}
    </code>
  );
}
