import { Rocket } from "lucide-react";

export function StepWelcome() {
  return (
    <div className="flex flex-col items-center text-center" data-testid="step-welcome">
      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-500/20 text-emerald-400 sm:h-16 sm:w-16">
        <Rocket className="h-6 w-6 sm:h-8 sm:w-8" aria-hidden="true" />
      </div>
      <h1 className="mt-4 text-2xl font-semibold sm:mt-6 sm:text-3xl">Welcome to Vrooli</h1>
      <p className="mt-2 max-w-md text-base text-slate-200 sm:mt-3 sm:text-lg">
        This wizard will guide you through configuring the resources that power
        your Vrooli installation. In just a few steps you will select the
        services you need, review your configuration, and generate a ready-to-use
        config file.
      </p>
      <div className="mt-5 grid w-full max-w-md gap-3 text-left sm:mt-8 sm:gap-4">
        <InfoCard
          number={1}
          title="Select Resources"
          description="Choose the AI, storage, and dev resources you want to enable."
        />
        <InfoCard
          number={2}
          title="Review Configuration"
          description="Validate your selections and see any warnings before proceeding."
        />
        <InfoCard
          number={3}
          title="Generate Config"
          description="Get a ready-to-use configuration file for your Vrooli server."
        />
      </div>
      <p className="mt-5 text-sm text-slate-300 sm:mt-8">
        Click <strong>Get Started</strong> below to begin.
      </p>
    </div>
  );
}

function InfoCard({ number, title, description }: { number: number; title: string; description: string }) {
  return (
    <div className="flex items-start gap-3 rounded-xl border border-white/10 bg-white/5 p-3 sm:gap-4 sm:p-4">
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-white/10 text-xs font-medium sm:h-8 sm:w-8 sm:text-sm">
        {number}
      </div>
      <div>
        <p className="text-sm font-medium sm:text-base">{title}</p>
        <p className="mt-0.5 text-xs text-slate-300 sm:mt-1 sm:text-sm">{description}</p>
      </div>
    </div>
  );
}
