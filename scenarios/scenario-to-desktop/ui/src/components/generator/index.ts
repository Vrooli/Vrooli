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

// Extracted components from GeneratorForm decomposition
export { AppMetadataSection, type AppMetadataSectionProps } from "./AppMetadataSection";
export { ConnectionSectionRouter, type ConnectionSectionRouterProps } from "./ConnectionSectionRouter";
export { GeneratorFormHeader, type GeneratorFormHeaderProps } from "./GeneratorFormHeader";
export { GeneratorFormFooter, type GeneratorFormFooterProps } from "./GeneratorFormFooter";
export { GeneratorModalsContainer, type GeneratorModalsContainerProps } from "./GeneratorModalsContainer";

// Re-export domain types for convenience
export { validateFormInputs, type ValidationError, type ValidateFormInputsParams } from "../../domain/generator";
