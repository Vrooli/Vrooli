/**
 * Node Field Components
 *
 * Reusable form field components for workflow node UIs.
 * These components integrate with useSyncedField hooks for
 * consistent state management and minimal boilerplate.
 *
 * @example
 * ```tsx
 * import {
 *   NodeTextField,
 *   NodeNumberField,
 *   NodeSelectField,
 *   NodeCheckbox,
 *   FieldRow,
 * } from './fields';
 * import { useSyncedString, useSyncedNumber, useSyncedBoolean } from '@hooks/useSyncedField';
 *
 * const MyNode = ({ id }) => {
 *   const { params, updateParams } = useActionParams<MyParams>(id);
 *
 *   const name = useSyncedString(params?.name ?? '', {
 *     onCommit: (v) => updateParams({ name: v || undefined }),
 *   });
 *   const count = useSyncedNumber(params?.count ?? 0, {
 *     min: 0,
 *     onCommit: (v) => updateParams({ count: v }),
 *   });
 *
 *   return (
 *     <BaseNode ...>
 *       <FieldRow>
 *         <NodeTextField field={name} label="Name" />
 *         <NodeNumberField field={count} label="Count" min={0} />
 *       </FieldRow>
 *     </BaseNode>
 *   );
 * };
 * ```
 */

export { NodeTextField } from './NodeTextField';
export { NodeTextArea } from './NodeTextArea';
export { NodeNumberField } from './NodeNumberField';
export { NodeSelectField } from './NodeSelectField';
export { NodeCheckbox } from './NodeCheckbox';
export { NodeUrlField } from './NodeUrlField';
export { NodeSelectorField } from './NodeSelectorField';
export { FieldRow } from './FieldRow';
export { TimeoutFields } from './TimeoutFields';

export type {
  BaseFieldProps,
  TextFieldProps,
  TextAreaProps,
  NumberFieldProps,
  SelectOption,
  SelectFieldProps,
  CheckboxProps,
  FieldRowProps,
} from './types';

export type { NodeUrlFieldProps } from './NodeUrlField';
export type { NodeSelectorFieldProps } from './NodeSelectorField';
export type { TimeoutFieldsProps } from './TimeoutFields';
