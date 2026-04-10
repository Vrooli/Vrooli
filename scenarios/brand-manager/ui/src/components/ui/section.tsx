import { cn } from "../../lib/utils";

interface SectionProps {
  title?: string;
  children: React.ReactNode;
  className?: string;
  /** data-testid for the section wrapper */
  testId?: string;
}

/**
 * Consistent card-like section used across all pages.
 * Owns the visual contract for grouped content blocks.
 */
export function Section({ title, children, className, testId }: SectionProps) {
  return (
    <section
      className={cn("rounded-xl border border-white/10 bg-white/5 p-5", className)}
      data-testid={testId}
    >
      {title && (
        <h2 className="text-sm font-medium text-slate-400 mb-3">{title}</h2>
      )}
      {children}
    </section>
  );
}
