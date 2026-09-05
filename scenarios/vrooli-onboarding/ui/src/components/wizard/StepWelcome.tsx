import { Rocket } from "lucide-react";

export function StepWelcome() {
  return (
    <div className="flex flex-col items-center text-center" data-testid="step-welcome">
      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/20 text-primary sm:h-16 sm:w-16">
        <Rocket className="h-6 w-6 sm:h-8 sm:w-8" aria-hidden="true" />
      </div>
      <h1 className="mt-4 text-2xl font-semibold sm:mt-6 sm:text-3xl">Welcome to Vrooli</h1>
      <p className="mt-2 max-w-md text-base text-foreground sm:mt-3 sm:text-lg">
        This wizard will guide you through configuring the capabilities that
        power your Vrooli installation. You will choose scenarios, review the
        manifest-derived resources and host permissions, apply the selection,
        and verify the resulting installation.
      </p>
      <div className="mt-5 grid w-full max-w-md gap-3 text-left sm:mt-8 sm:gap-4">
        <InfoCard
          number={1}
          title="Select Scenarios"
          description="Choose the capabilities this installation should run."
        />
        <InfoCard
          number={2}
          title="Review Permissions"
          description="Inspect derived resources, credentials, and host changes before consent."
        />
        <InfoCard
          number={3}
          title="Apply and Verify"
          description="Apply the committed selection and confirm readiness with live checks."
        />
      </div>
      <p className="mt-5 text-sm text-muted sm:mt-8">
        Click <strong>Get Started</strong> below to begin.
      </p>
    </div>
  );
}

function InfoCard({ number, title, description }: { number: number; title: string; description: string }) {
  return (
    <div className="flex items-start gap-3 rounded-xl border border-muted bg-surface-muted p-3 sm:gap-4 sm:p-4">
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-surface-subtle text-xs font-medium sm:h-8 sm:w-8 sm:text-sm">
        {number}
      </div>
      <div>
        <p className="text-sm font-medium sm:text-base">{title}</p>
        <p className="mt-0.5 text-xs text-muted sm:mt-1 sm:text-sm">{description}</p>
      </div>
    </div>
  );
}
