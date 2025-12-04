import { CheckCircle2, Dot, FileOutput, Info, Map, Rocket } from "lucide-react";
import { Button } from "../components/ui/button";

interface DeploymentStepperProps {
  activeStep: number;
  onStepChange: (index: number) => void;
  onOpenResource?: () => void;
  onGenerateManifest?: () => void;
  hasManifest?: boolean;
}

const steps = [
  { title: "Manifest primer", icon: Info, description: "Understand what goes into a deployment manifest." },
  { title: "Check readiness", icon: Map, description: "Verify tiers and strategies for your campaign." },
  { title: "Fix blockers", icon: Rocket, description: "Jump to the right resource to define strategies." },
  { title: "Export manifest", icon: FileOutput, description: "Generate and download the bundle." }
];

export const DeploymentStepper = ({
  activeStep,
  onStepChange,
  onOpenResource,
  onGenerateManifest,
  hasManifest
}: DeploymentStepperProps) => {
  return (
    <section className="rounded-3xl border border-white/10 bg-white/5 p-6">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Campaign Steps</p>
          <p className="text-lg font-semibold text-white">Skip around as needed</p>
          <p className="text-sm text-white/60">
            Start with the primer, then hop to readiness, blockers, and export without losing your place.
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs text-white/60">
          <CheckCircle2 className="h-4 w-4 text-emerald-300" />
          <span>{hasManifest ? "Manifest generated" : "Manifest not generated yet"}</span>
        </div>
      </div>
      <div className="mt-4 flex flex-wrap gap-3">
        {steps.map((step, index) => {
          const Icon = step.icon;
          const isActive = index === activeStep;
          const isComplete = index < activeStep;
          return (
            <button
              key={step.title}
              className={`group flex flex-1 min-w-[240px] items-center gap-3 rounded-2xl border px-3 py-3 text-left transition ${
                isActive
                  ? "border-emerald-400 bg-emerald-500/10 shadow-[0_10px_40px_-15px_rgba(16,185,129,0.4)]"
                  : "border-white/10 bg-black/20 hover:border-white/30"
              }`}
              onClick={() => onStepChange(index)}
            >
              <div className="rounded-full border border-white/15 bg-white/5 p-2">
                {isComplete ? <CheckCircle2 className="h-4 w-4 text-emerald-300" /> : <Icon className="h-4 w-4 text-white/60" />}
              </div>
              <div className="flex-1">
                <p className="text-xs uppercase tracking-[0.2em] text-white/60">{step.title}</p>
                <p className="text-sm text-white/80">{step.description}</p>
              </div>
              <Dot className={`h-5 w-5 ${isActive ? "text-emerald-300" : "text-white/40"}`} />
            </button>
          );
        })}
      </div>
      <div className="mt-4 flex flex-wrap gap-2 text-sm text-white/70">
        <Button size="sm" variant="outline" onClick={() => onStepChange(0)}>
          Primer
        </Button>
        <Button size="sm" variant="outline" onClick={() => onStepChange(1)}>
          Jump to readiness
        </Button>
        <Button size="sm" variant="outline" onClick={() => onOpenResource?.()}>
          Open blocker
        </Button>
        <Button size="sm" variant="secondary" onClick={() => onGenerateManifest?.()}>
          Generate manifest
        </Button>
      </div>
    </section>
  );
};
