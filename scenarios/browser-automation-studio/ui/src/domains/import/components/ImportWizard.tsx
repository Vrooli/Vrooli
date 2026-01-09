/**
 * ImportWizard Component
 *
 * Multi-step wizard container for import operations.
 * Provides step navigation, progress indication, and validation.
 */

import { useState, useCallback, useEffect, useId } from 'react';
import { ChevronLeft, ChevronRight, Loader2, X } from 'lucide-react';
import { ResponsiveDialog } from '@shared/layout';
import type { ImportWizardStep } from '../types';

export interface ImportWizardProps {
  /** Whether the wizard is open */
  isOpen: boolean;
  /** Callback when wizard is closed */
  onClose: () => void;
  /** Wizard steps */
  steps: ImportWizardStep[];
  /** Callback when wizard is completed */
  onComplete: () => void;
  /** Whether completion is in progress */
  isLoading?: boolean;
  /** Title for the wizard */
  title: string;
  /** Subtitle/description */
  subtitle?: string;
  /** Icon component for the header */
  icon?: React.ReactNode;
  /** Label for the complete button */
  completeLabel?: string;
  /** Current step content renderer */
  children: (stepId: string) => React.ReactNode;
  /** Initial step index */
  initialStep?: number;
  /** Custom class name */
  className?: string;
}

export function ImportWizard({
  isOpen,
  onClose,
  steps,
  onComplete,
  isLoading = false,
  title,
  subtitle,
  icon,
  completeLabel = 'Import',
  children,
  initialStep = 0,
  className = '',
}: ImportWizardProps) {
  const [currentStep, setCurrentStep] = useState(initialStep);
  const titleId = useId();

  // Reset to initial step when wizard opens
  useEffect(() => {
    if (isOpen) {
      setCurrentStep(initialStep);
    }
  }, [isOpen, initialStep]);

  const currentStepData = steps[currentStep];
  const isFirstStep = currentStep === 0;
  const isLastStep = currentStep === steps.length - 1;

  const handleNext = useCallback(() => {
    if (currentStep < steps.length - 1) {
      setCurrentStep((prev) => prev + 1);
    }
  }, [currentStep, steps.length]);

  const handleBack = useCallback(() => {
    if (currentStep > 0) {
      setCurrentStep((prev) => prev - 1);
    }
  }, [currentStep]);

  const handleComplete = useCallback(() => {
    if (currentStepData?.isValid) {
      onComplete();
    }
  }, [currentStepData, onComplete]);

  const canProceed = currentStepData?.isValid ?? false;

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="default"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className={`bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden ${className}`}
    >
      <div>
        {/* Header */}
        <div className="relative px-6 pt-6 pb-4">
          {/* Close button */}
          <button
            type="button"
            onClick={onClose}
            className="absolute top-4 right-4 p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close"
            disabled={isLoading}
          >
            <X size={18} />
          </button>

          {/* Icon & Title */}
          <div className="flex flex-col items-center text-center mb-2">
            {icon && (
              <div className="w-14 h-14 bg-gradient-to-br from-flow-accent/30 to-blue-500/20 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-flow-accent/10">
                {icon}
              </div>
            )}
            <h2 id={titleId} className="text-xl font-bold text-white tracking-tight">
              {title}
            </h2>
            {subtitle && <p className="text-sm text-gray-400 mt-1">{subtitle}</p>}
          </div>

          {/* Step Indicator */}
          {steps.length > 1 && (
            <div className="flex items-center justify-center gap-2 mt-4">
              {steps.map((step, index) => (
                <div key={step.id} className="flex items-center">
                  <div
                    className={`
                      w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors
                      ${
                        index === currentStep
                          ? 'bg-flow-accent text-white'
                          : index < currentStep
                          ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                          : 'bg-gray-700/50 text-gray-500'
                      }
                    `}
                    title={step.title}
                  >
                    {index + 1}
                  </div>
                  {index < steps.length - 1 && (
                    <div
                      className={`w-8 h-0.5 mx-1 ${
                        index < currentStep ? 'bg-green-500/30' : 'bg-gray-700/50'
                      }`}
                    />
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Step Title */}
        {currentStepData && (
          <div className="px-6 pb-2">
            <h3 className="text-sm font-medium text-gray-300">{currentStepData.title}</h3>
            {currentStepData.description && (
              <p className="text-xs text-gray-500 mt-0.5">{currentStepData.description}</p>
            )}
          </div>
        )}

        {/* Content */}
        <div className="px-6 pb-4 min-h-[200px]">
          {currentStepData && children(currentStepData.id)}
        </div>

        {/* Actions */}
        <div className="px-6 pb-6 flex items-center justify-between">
          <div>
            {!isFirstStep && (
              <button
                type="button"
                onClick={handleBack}
                disabled={isLoading}
                className="flex items-center gap-1 px-4 py-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors disabled:opacity-50"
              >
                <ChevronLeft size={16} />
                Back
              </button>
            )}
          </div>

          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors disabled:opacity-50"
            >
              Cancel
            </button>

            {isLastStep ? (
              <button
                type="button"
                onClick={handleComplete}
                disabled={!canProceed || isLoading}
                className="px-5 py-2.5 bg-gradient-to-r from-green-500 to-emerald-600 text-white font-medium rounded-xl hover:from-green-400 hover:to-emerald-500 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-green-500/20 hover:shadow-green-500/30"
              >
                {isLoading ? (
                  <span className="flex items-center gap-2">
                    <Loader2 size={16} className="animate-spin" />
                    Processing...
                  </span>
                ) : (
                  completeLabel
                )}
              </button>
            ) : (
              <button
                type="button"
                onClick={handleNext}
                disabled={!canProceed || isLoading}
                className="flex items-center gap-1 px-5 py-2.5 bg-gradient-to-r from-flow-accent to-blue-600 text-white font-medium rounded-xl hover:from-blue-500 hover:to-blue-700 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-flow-accent/20"
              >
                Next
                <ChevronRight size={16} />
              </button>
            )}
          </div>
        </div>
      </div>
    </ResponsiveDialog>
  );
}

export default ImportWizard;
