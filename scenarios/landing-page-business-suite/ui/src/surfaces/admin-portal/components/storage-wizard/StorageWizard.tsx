import { useEffect } from 'react';
import { ArrowLeft, ArrowRight, RefreshCw } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import { Stepper, type Step } from '../../../../shared/ui/stepper';
import { Callout } from '../Callout';
import { useStorageWizard } from '../../hooks/useStorageWizard';
import { StepProvider } from './StepProvider';
import { StepConfiguration } from './StepConfiguration';
import { StepCredentials } from './StepCredentials';
import { StepVerify } from './StepVerify';
import type { DownloadStorageSettingsSnapshot } from '../../../../shared/api';

const WIZARD_STEPS: Step[] = [
  { id: 'provider', label: 'Provider' },
  { id: 'configure', label: 'Configure' },
  { id: 'credentials', label: 'Credentials' },
  { id: 'verify', label: 'Verify' },
];

interface StorageWizardProps {
  initialSettings: DownloadStorageSettingsSnapshot | null;
  onComplete: () => void;
}

export function StorageWizard({ onComplete }: StorageWizardProps) {
  const wizard = useStorageWizard({ onComplete });
  const { loadExistingSettings } = wizard;

  useEffect(() => {
    void loadExistingSettings();
  }, [loadExistingSettings]);

  const renderStep = () => {
    switch (wizard.currentStepId) {
      case 'provider':
        return (
          <StepProvider
            selectedProvider={wizard.state.provider}
            onSelectProvider={wizard.setProvider}
          />
        );
      case 'configure':
        if (!wizard.state.provider) {
          return null;
        }
        return (
          <StepConfiguration
            provider={wizard.state.provider}
            form={wizard.state.form}
            cloudflareAccountId={wizard.state.cloudflareAccountId}
            onFormChange={wizard.setForm}
            onCloudflareAccountIdChange={wizard.setCloudflareAccountId}
          />
        );
      case 'credentials':
        if (!wizard.state.provider) {
          return null;
        }
        return (
          <StepCredentials
            provider={wizard.state.provider}
            credentials={wizard.state.credentials}
            existingSettings={wizard.state.existingSettings}
            onCredentialsChange={wizard.setCredentials}
          />
        );
      case 'verify':
        if (!wizard.state.provider) {
          return null;
        }
        return (
          <StepVerify
            provider={wizard.state.provider}
            form={wizard.state.form}
            cloudflareAccountId={wizard.state.cloudflareAccountId}
            testStatus={wizard.state.testStatus}
            testError={wizard.state.testError}
            saveStatus={wizard.state.saveStatus}
            saveError={wizard.state.saveError}
            onFormChange={wizard.setForm}
            onTestConnection={wizard.testConnection}
            onSave={wizard.saveSettings}
          />
        );
      default:
        return null;
    }
  };

  if (wizard.state.loading) {
    return (
      <div className="rounded-2xl border border-white/10 bg-white/5 p-8">
        <div className="flex flex-col items-center justify-center gap-4">
          <RefreshCw className="h-8 w-8 animate-spin text-blue-400" />
          <p className="text-slate-400">Loading storage settings...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-2xl border border-white/10 bg-white/5 p-6 space-y-6">
      {/* Stepper */}
      <Stepper
        steps={WIZARD_STEPS}
        currentStep={wizard.state.step}
        onStepClick={(index) => {
          // Only allow clicking on completed steps or current step
          if (index <= wizard.state.step) {
            wizard.goToStep(index);
          }
        }}
      />

      {/* Load error */}
      {wizard.state.loadError && (
        <Callout type="error" message={wizard.state.loadError} />
      )}

      {/* Step Content */}
      <div className="min-h-[300px]">{renderStep()}</div>

      {/* Navigation */}
      {wizard.currentStepId !== 'verify' && (
        <div className="flex items-center justify-between pt-4 border-t border-white/10">
          <Button
            variant="outline"
            onClick={wizard.goBack}
            disabled={!wizard.canGoBack}
            className="gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>

          <Button
            onClick={wizard.goNext}
            disabled={!wizard.canGoNext}
            className="gap-2"
          >
            Next
            <ArrowRight className="h-4 w-4" />
          </Button>
        </div>
      )}

      {/* Back button on verify step (save is in the step itself) */}
      {wizard.currentStepId === 'verify' && (
        <div className="flex items-center pt-4 border-t border-white/10">
          <Button
            variant="outline"
            onClick={wizard.goBack}
            className="gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
        </div>
      )}
    </div>
  );
}
