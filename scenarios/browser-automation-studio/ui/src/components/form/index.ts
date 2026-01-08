/**
 * Form Components
 *
 * Reusable form input components for settings and configuration UIs.
 * These components use a controlled pattern (value + onChange) and
 * support both light and dark themes.
 *
 * @example
 * ```tsx
 * import {
 *   FormNumberInput,
 *   FormTextInput,
 *   FormSelect,
 *   FormCheckbox,
 *   FormFieldGroup,
 *   FormGrid,
 * } from '@/components/form';
 *
 * function SettingsForm({ settings, onChange }) {
 *   return (
 *     <FormFieldGroup title="Display" description="Configure display options">
 *       <FormGrid cols={2}>
 *         <FormNumberInput
 *           value={settings.width}
 *           onChange={(v) => onChange('width', v)}
 *           label="Width"
 *           placeholder="1920"
 *         />
 *         <FormNumberInput
 *           value={settings.height}
 *           onChange={(v) => onChange('height', v)}
 *           label="Height"
 *           placeholder="1080"
 *         />
 *       </FormGrid>
 *     </FormFieldGroup>
 *   );
 * }
 * ```
 */

export { FormNumberInput } from './FormNumberInput';
export { FormTextInput } from './FormTextInput';
export { FormSelect } from './FormSelect';
export { FormCheckbox } from './FormCheckbox';
export { FormFieldGroup } from './FormFieldGroup';
export { FormGrid } from './FormGrid';

export type { FormNumberInputProps } from './FormNumberInput';
export type { FormTextInputProps } from './FormTextInput';
export type { FormSelectProps, FormSelectOption } from './FormSelect';
export type { FormCheckboxProps } from './FormCheckbox';
export type { FormFieldGroupProps } from './FormFieldGroup';
export type { FormGridProps } from './FormGrid';

// Style constants for custom components
export {
  FORM_INPUT_CLASSES,
  FORM_SELECT_CLASSES,
  FORM_LABEL_CLASSES,
  FORM_DESCRIPTION_CLASSES,
  FORM_CHECKBOX_CLASSES,
} from './styles';
