/**
 * Shared types for node field components.
 *
 * These components provide a consistent, declarative API for building
 * node forms with minimal boilerplate.
 */

import type { ComponentType } from 'react';
import type { LucideIcon } from 'lucide-react';
import type { UseSyncedFieldResult } from '@hooks/useSyncedField';

// ============================================================================
// Base Props
// ============================================================================

export interface BaseFieldProps {
  /** Label text shown above the field */
  label: string;
  /** Optional description text below the field */
  description?: string;
  /** Additional className for the wrapper div */
  className?: string;
  /** Icon component to show next to label (supports Lucide icons) */
  icon?: LucideIcon | ComponentType<{ size?: number | string; className?: string }>;
  /** Icon color class (e.g., "text-blue-400") */
  iconClassName?: string;
}

// ============================================================================
// Text Fields
// ============================================================================

export interface TextFieldProps extends BaseFieldProps {
  /** Field state from useSyncedString */
  field: UseSyncedFieldResult<string>;
  /** Placeholder text */
  placeholder?: string;
  /** Whether the field is disabled */
  disabled?: boolean;
}

export interface TextAreaProps extends TextFieldProps {
  /** Number of visible text rows */
  rows?: number;
}

// ============================================================================
// Number Fields
// ============================================================================

export interface NumberFieldProps extends BaseFieldProps {
  /** Field state from useSyncedNumber */
  field: UseSyncedFieldResult<number>;
  /** Minimum allowed value */
  min?: number;
  /** Maximum allowed value */
  max?: number;
  /** Step increment for arrows/scroll */
  step?: number;
  /** Placeholder text */
  placeholder?: string;
  /** Whether the field is disabled */
  disabled?: boolean;
}

// ============================================================================
// Select Fields
// ============================================================================

export interface SelectOption<T extends string = string> {
  value: T;
  label: string;
}

export interface SelectFieldProps<T extends string = string> extends BaseFieldProps {
  /** Field state from useSyncedSelect */
  field: UseSyncedFieldResult<T>;
  /** Available options */
  options: SelectOption<T>[];
  /** Whether the field is disabled */
  disabled?: boolean;
}

// ============================================================================
// Checkbox
// ============================================================================

export interface CheckboxProps {
  /** Field state from useSyncedBoolean */
  field: UseSyncedFieldResult<boolean>;
  /** Label text shown next to checkbox */
  label: string;
  /** Additional className */
  className?: string;
  /** Whether the checkbox is disabled */
  disabled?: boolean;
}

// ============================================================================
// Layout
// ============================================================================

export interface FieldRowProps {
  /** Child field components */
  children: React.ReactNode;
  /** Number of columns (default: 2) */
  cols?: 2 | 3 | 4;
  /** Additional className */
  className?: string;
}
