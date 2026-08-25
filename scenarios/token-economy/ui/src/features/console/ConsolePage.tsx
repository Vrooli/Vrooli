import type { FormEventHandler, ReactNode } from "react";
import { AlertCircle, CheckCircle2, LoaderCircle } from "lucide-react";

import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { ExperienceSurface } from "../../components/experience/ExperienceSurface";

export function ConsolePage({
  testId,
  title,
  description,
  children,
}: {
  testId: string;
  title: string;
  description: string;
  children: ReactNode;
}) {
  const headingId = `${testId}-heading`;
  return (
    <ExperienceSurface surfaceId={testId} state="ready" data-testid={testId} aria-labelledby={headingId} className="flex min-w-0 flex-col gap-5">
      <header className="border-b border-app-border pb-4">
        <h2 id={headingId} className="text-2xl font-semibold text-app-foreground">{title}</h2>
        <p className="mt-1 max-w-3xl text-sm text-app-muted-foreground">{description}</p>
      </header>
      {children}
    </ExperienceSurface>
  );
}

export function ConsoleForm({
  title,
  submitLabel,
  submitTestId,
  busy,
  onSubmit,
  children,
}: {
  title: string;
  submitLabel: string;
  submitTestId: string;
  busy: boolean;
  onSubmit: FormEventHandler<HTMLFormElement>;
  children: ReactNode;
}) {
  return (
    <Card>
      <CardHeader><CardTitle>{title}</CardTitle></CardHeader>
      <CardContent>
        <form className="grid gap-4 md:grid-cols-2 xl:grid-cols-3" onSubmit={onSubmit}>
          {children}
          <div className="flex items-end">
            <Button data-testid={submitTestId} type="submit" disabled={busy} className="w-full md:w-auto">
              {busy ? <LoaderCircle aria-hidden className="h-4 w-4 animate-spin motion-reduce:animate-none" /> : <CheckCircle2 aria-hidden className="h-4 w-4" />}
              {submitLabel}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return <label className="grid gap-1 text-sm font-medium text-app-foreground"><span>{label}</span>{children}</label>;
}

export function RequestState({
  loading,
  error,
  loadingLabel,
  errorLabel,
}: {
  loading: boolean;
  error: unknown;
  loadingLabel: string;
  errorLabel: string;
}) {
  if (loading) {
    return <p role="status" className="inline-flex items-center gap-2 text-sm text-app-muted-foreground"><LoaderCircle aria-hidden className="h-4 w-4 animate-spin motion-reduce:animate-none" />{loadingLabel}</p>;
  }
  if (error) {
    return <p role="alert" className="inline-flex items-center gap-2 rounded-control border border-app-danger/30 bg-app-danger/10 p-3 text-sm text-app-danger"><AlertCircle aria-hidden className="h-4 w-4" />{errorLabel}</p>;
  }
  return null;
}
