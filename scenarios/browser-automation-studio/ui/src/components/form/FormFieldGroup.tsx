import type { FC, ReactNode } from 'react';

export interface FormFieldGroupProps {
  /** Section title */
  title: string;
  /** Optional description text below the title */
  description?: string;
  /** Child form fields */
  children: ReactNode;
  /** Additional CSS classes for the wrapper */
  className?: string;
}

/**
 * Groups form fields under a section header with consistent styling.
 *
 * @example
 * ```tsx
 * <FormFieldGroup title="Viewport" description="Browser window dimensions">
 *   <FormGrid cols={2}>
 *     <FormNumberInput label="Width" ... />
 *     <FormNumberInput label="Height" ... />
 *   </FormGrid>
 * </FormFieldGroup>
 * ```
 */
export const FormFieldGroup: FC<FormFieldGroupProps> = ({
  title,
  description,
  children,
  className,
}) => (
  <div className={className}>
    <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-1">{title}</h4>
    {description && (
      <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">{description}</p>
    )}
    {!description && <div className="mb-3" />}
    {children}
  </div>
);
