/**
 * Framework and template selection section for the generator form.
 */

import { Button } from "../ui/button";
import { Label } from "../ui/label";
import {
  TEMPLATE_SUMMARIES,
  FRAMEWORK_SUMMARIES,
} from "../../domain/generator";
import { selectors } from "../../consts/selectors";

export interface FrameworkTemplateSectionProps {
  framework: string;
  onOpenFrameworkModal: () => void;
  selectedTemplate: string;
  onOpenTemplateModal: () => void;
}

export function FrameworkTemplateSection({
  framework,
  onOpenFrameworkModal,
  selectedTemplate,
  onOpenTemplateModal,
}: FrameworkTemplateSectionProps) {
  const frameworkSummary = FRAMEWORK_SUMMARIES[framework] ?? {
    name: framework,
    description: "Desktop framework",
  };
  const templateSummary = TEMPLATE_SUMMARIES[selectedTemplate] ?? {
    name: selectedTemplate.replace(/_/g, " "),
    description: "Custom template",
  };

  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div>
        <Label>Framework</Label>
        <div className="mt-1.5 rounded-md border border-slate-800 bg-slate-950/60 p-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-sm font-semibold text-slate-100">
                {frameworkSummary.name}
              </p>
              <p className="text-xs text-slate-400">
                {frameworkSummary.description}
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onOpenFrameworkModal}
            >
              Browse frameworks
            </Button>
          </div>
        </div>
        <p className="mt-1.5 text-xs text-slate-400">
          Electron is the supported desktop framework.
        </p>
      </div>

      <div>
        <Label>Template</Label>
        <div className="mt-1.5 rounded-md border border-slate-800 bg-slate-950/60 p-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-sm font-semibold text-slate-100">
                {templateSummary.name}
              </p>
              <p className="text-xs text-slate-400">
                {templateSummary.description}
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onOpenTemplateModal}
              data-testid={selectors.generator.templatePicker}
            >
              Browse templates
            </Button>
          </div>
        </div>
        <p className="mt-1.5 text-xs text-slate-400">
          All templates share the same codebase. If you change your mind later,
          switch templates here or from the Generated Apps tab - your scenario
          stays intact.
        </p>
      </div>
    </div>
  );
}
