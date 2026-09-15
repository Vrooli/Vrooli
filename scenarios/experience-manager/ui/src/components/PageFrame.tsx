import type { ReactNode } from "react";

export function PageFrame({
  testId,
  title,
  description,
  experienceSurface,
  experienceState,
  children,
}: {
  testId: string;
  title: string;
  description: string;
  experienceSurface?: string;
  experienceState?: "loading" | "ready" | "empty" | "partial" | "error";
  children: ReactNode;
}) {
  return (
    <section data-testid={testId} data-experience-surface={experienceSurface} data-experience-state={experienceState} aria-live={experienceSurface ? "polite" : undefined} aria-labelledby={`${testId}-heading`} className="flex min-w-0 flex-col gap-5">
      <div className="flex flex-col gap-2">
        <h2 id={`${testId}-heading`} className="text-2xl font-semibold text-app-foreground">
          {title}
        </h2>
        <p className="max-w-3xl text-sm text-app-muted-foreground">{description}</p>
      </div>
      {children}
    </section>
  );
}
