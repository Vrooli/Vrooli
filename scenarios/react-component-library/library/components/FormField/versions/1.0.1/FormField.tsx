/**
 * @libraryId react-component-library:FormField
 * @displayName FormField
 * @description A complete accessible field composition with stable label, description, control, and error relationships.
 * @version 1.0.1
 * @tags ["form","accessibility","token-bound","compound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource forms.form-field */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import {
  cloneElement,
  isValidElement,
  useId,
  type CSSProperties,
  type ReactElement,
  type ReactNode,
} from "react";

export interface FormFieldProps {
  label: ReactNode;
  control: ReactElement<ControlElementProps>;
  id?: string;
  description?: ReactNode;
  error?: ReactNode;
  required?: boolean;
  disabled?: boolean;
  optionalLabel?: ReactNode;
  className?: string;
  style?: CSSProperties;
}

interface ControlElementProps {
  id?: string;
  disabled?: boolean;
  required?: boolean;
  "aria-invalid"?: boolean | string;
  "aria-describedby"?: string;
  [key: string]: unknown;
}

const styles = `
  [data-rcl-form-field] { display: grid; gap: var(--space-2xs, .5rem); color: var(--color-foreground, #0f172a); }
  [data-rcl-form-field-label-row] { display: flex; align-items: baseline; justify-content: space-between; gap: var(--space-sm, .75rem); }
  [data-rcl-form-field-label] { color: var(--color-foreground, #0f172a); font: var(--text-label, 600 .8125rem/1.25rem system-ui, sans-serif); letter-spacing: var(--text-label-tracking, .01em); }
  [data-rcl-form-field-required] { color: var(--color-danger, #dc2626); margin-inline-start: var(--space-3xs, .25rem); }
  [data-rcl-form-field-optional] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 500 .75rem/1rem system-ui, sans-serif); }
  [data-rcl-form-field-description] { color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 .875rem/1.375rem system-ui, sans-serif); }
  [data-rcl-form-field-error] { display: flex; align-items: flex-start; gap: var(--space-2xs, .5rem); color: var(--color-danger, #dc2626); font: var(--text-body, 400 .875rem/1.375rem system-ui, sans-serif); }
  [data-rcl-form-field-error-mark] { display: inline-grid; place-items: center; flex: 0 0 auto; inline-size: 1.125rem; block-size: 1.125rem; margin-block-start: .125rem; border: 1px solid currentColor; border-radius: 50%; font: 700 .6875rem/1 system-ui, sans-serif; }
  [data-rcl-form-field][data-invalid="true"] [data-rcl-form-field-control] { --rcl-field-invalid: var(--color-danger, #dc2626); }
  [data-rcl-form-field][data-disabled="true"] { opacity: var(--opacity-disabled, .58); }
  [data-rcl-form-field-control] { min-inline-size: 0; }
`;

function mergeDescribedBy(...ids: Array<string | undefined>) {
  return ids.filter(Boolean).join(" ") || undefined;
}

export const FormField = withClassName(function FormField({
  label,
  control,
  id,
  description,
  error,
  required = false,
  disabled = false,
  optionalLabel = "Optional",
  className,
  style,
}: FormFieldProps) {
  const generatedId = useId().replace(/:/g, "");
  const controlId = id ?? `rcl-form-field-${generatedId}`;
  const descriptionId = description ? `${controlId}-description` : undefined;
  const errorId = error ? `${controlId}-error` : undefined;
  const describedBy = mergeDescribedBy(descriptionId, errorId);
  const enhancedControl = isValidElement(control)
    ? cloneElement(control, {
        id: controlId,
        disabled: disabled || control.props.disabled,
        required: required || control.props.required,
        "aria-invalid": error ? true : control.props["aria-invalid"],
        "aria-describedby": mergeDescribedBy(control.props["aria-describedby"], describedBy),
        "data-rcl-form-field-control": "true",
      })
    : control;

  return (
    <>
      <StyleSheet name="form-field-1-0-1" css={styles} />
      <div
        data-testid="forms.form-field"
        className={className}
        data-rcl-form-field="true"
        data-invalid={Boolean(error)}
        data-disabled={disabled}
        style={style}
      >
        <div data-rcl-form-field-label-row>
          <label data-rcl-form-field-label htmlFor={controlId}>
            {label}
            {required && (
              <span data-rcl-form-field-required aria-hidden="true">
                *
              </span>
            )}
          </label>
          {!required && <span data-rcl-form-field-optional>{optionalLabel}</span>}
        </div>
        <div data-rcl-form-field-control-slot>{enhancedControl}</div>
        {description && (
          <div id={descriptionId} data-rcl-form-field-description>
            {description}
          </div>
        )}
        {error && (
          <div id={errorId} data-rcl-form-field-error role="alert">
            <span data-rcl-form-field-error-mark aria-hidden="true">
              !
            </span>
            <span>{error}</span>
          </div>
        )}
      </div>
    </>
  );
});
