import { ChevronLeft, ChevronRight, RotateCcw, Home } from "lucide-react";
import { Stepper } from "../ui/stepper";
import { Button } from "../ui/button";
import { Card, CardContent } from "../ui/card";
import { useDeployment } from "../../hooks/useDeployment";
import { StepManifest } from "./StepManifest";
import { StepValidate } from "./StepValidate";
import { StepPlan } from "./StepPlan";
import { StepBuild } from "./StepBuild";
import { StepPreflight } from "./StepPreflight";
import { StepDeploy } from "./StepDeploy";
import { cn } from "../../lib/utils";

interface WizardContainerProps {
  onBackToDashboard?: () => void;
}

export function WizardContainer({ onBackToDashboard }: WizardContainerProps) {
  const deployment = useDeployment();
  const { currentStepIndex, currentStep, steps, goToStep, goNext, goBack, canProceed, reset } = deployment;

  const renderStep = () => {
    switch (currentStep.id) {
      case "manifest":
        return <StepManifest deployment={deployment} />;
      case "validate":
        return <StepValidate deployment={deployment} />;
      case "plan":
        return <StepPlan deployment={deployment} />;
      case "build":
        return <StepBuild deployment={deployment} />;
      case "preflight":
        return <StepPreflight deployment={deployment} />;
      case "deploy":
        return <StepDeploy deployment={deployment} />;
      default:
        return null;
    }
  };

  const isFirstStep = currentStepIndex === 0;
  const isLastStep = currentStepIndex === steps.length - 1;

  return (
    <div className="space-y-6" data-testid="wizard-container">
      {/* Stepper */}
      <Card className="overflow-hidden">
        <CardContent className="py-6">
          <Stepper
            steps={steps}
            currentStep={currentStepIndex}
            onStepClick={goToStep}
            data-testid="wizard-stepper"
          />
        </CardContent>
      </Card>

      {/* Current Step Content */}
      <Card>
        <CardContent className="py-6">
          {/* Step Header */}
          <div className="mb-6">
            <div className="flex items-center gap-2">
              <span className="px-2 py-0.5 text-xs font-medium rounded bg-blue-500/20 text-blue-300">
                Step {currentStepIndex + 1} of {steps.length}
              </span>
            </div>
            <h2 className="mt-2 text-xl font-semibold text-white">
              {currentStep.label}
            </h2>
            <p className="mt-1 text-sm text-slate-400">
              {currentStep.description}
            </p>
          </div>

          {/* Step Content */}
          <div className="min-h-[300px]">
            {renderStep()}
          </div>
        </CardContent>
      </Card>

      {/* Navigation */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {onBackToDashboard && (
            <Button
              data-testid="wizard-dashboard-button"
              variant="outline"
              onClick={onBackToDashboard}
              className="text-slate-400 hover:text-slate-200"
            >
              <Home className="h-4 w-4 mr-1" />
              <span className="hidden sm:inline">Dashboard</span>
            </Button>
          )}
          <Button
            data-testid="wizard-back-button"
            variant="outline"
            onClick={goBack}
            disabled={isFirstStep}
            className={cn(isFirstStep && "invisible")}
          >
            <ChevronLeft className="h-4 w-4 mr-1" />
            Back
          </Button>
          <Button
            data-testid="wizard-reset-button"
            variant="outline"
            onClick={reset}
            className="text-slate-400 hover:text-slate-200"
          >
            <RotateCcw className="h-4 w-4 mr-1" />
            <span className="hidden sm:inline">Start Over</span>
          </Button>
        </div>

        {!isLastStep && (
          <Button
            data-testid="wizard-next-button"
            onClick={goNext}
            disabled={!canProceed}
          >
            Continue
            <ChevronRight className="h-4 w-4 ml-1" />
          </Button>
        )}
      </div>
    </div>
  );
}
