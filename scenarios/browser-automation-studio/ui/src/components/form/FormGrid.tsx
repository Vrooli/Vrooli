import type { FC, ReactNode } from 'react';

export interface FormGridProps {
  /** Number of columns in the grid */
  cols?: 2 | 3 | 4;
  /** Child form fields */
  children: ReactNode;
  /** Additional CSS classes */
  className?: string;
}

const GRID_CLASSES: Record<2 | 3 | 4, string> = {
  2: 'grid grid-cols-2 gap-4',
  3: 'grid grid-cols-3 gap-4',
  4: 'grid grid-cols-4 gap-4',
};

/**
 * Arranges form fields in a responsive grid layout.
 *
 * @example
 * ```tsx
 * <FormGrid cols={2}>
 *   <FormNumberInput label="Min Delay" ... />
 *   <FormNumberInput label="Max Delay" ... />
 * </FormGrid>
 * ```
 */
export const FormGrid: FC<FormGridProps> = ({ cols = 2, children, className }) => (
  <div className={`${GRID_CLASSES[cols]}${className ? ` ${className}` : ''}`}>{children}</div>
);
