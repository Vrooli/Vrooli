import type { ReactNode } from "react";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";

import { cn } from "../lib/utils";

/** The reading widths of the Hub: lists and records share one measure so pages do not jump. */
const MEASURE = {
  list: "max-w-[44rem]",
  wide: "max-w-[64rem]",
} as const;

/**
 * One routed page: its `h1`, optional description and actions, and a body at
 * the Hub's reading measure. Pages do not draw their own titles.
 */
export function Page({ title, description, actions, leading, eyebrow, headingId, testId, width = "list", children }: {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  leading?: ReactNode;
  eyebrow?: ReactNode;
  /** Id of the `h1`, so a page can move focus to it. */
  headingId: string;
  testId?: string;
  width?: keyof typeof MEASURE;
  children: ReactNode;
}) {
  return (
    <section data-testid={testId} aria-labelledby={headingId} className={cn("mx-auto flex w-full flex-col gap-4", MEASURE[width])}>
      {leading}
      <PageHeader title={title} description={description} actions={actions} eyebrow={eyebrow} headingId={headingId} />
      {children}
    </section>
  );
}
