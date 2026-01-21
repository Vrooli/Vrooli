/**
 * Generator form components.
 */

export { ScenarioSelector, type ScenarioSelectorProps } from "./ScenarioSelector";
export {
  FrameworkTemplateSection,
  type FrameworkTemplateSectionProps,
} from "./FrameworkTemplateSection";
export { SigningInlineSection, type SigningInlineSectionProps } from "./SigningInlineSection";
export {
  OutputLocationSelector,
  OutputPathField,
  type OutputLocationSelectorProps,
  type OutputPathFieldProps
} from "./OutputLocationSelector";
export {
  ValidationErrors,
  type ValidationErrorsProps,
} from "./ValidationErrors";

// Re-export constants from constants.ts
export { TEMPLATE_SUMMARIES, FRAMEWORK_SUMMARIES } from "./constants";

// Re-export domain types for convenience
export { validateFormInputs, type ValidationError, type ValidateFormInputsParams } from "../../domain/generator";
