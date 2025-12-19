import { FC } from 'react';
import type { FieldRowProps } from './types';

const colsClasses = {
  2: 'grid-cols-2',
  3: 'grid-cols-3',
  4: 'grid-cols-4',
} as const;

/**
 * Grid layout for placing fields side-by-side.
 *
 * @example
 * ```tsx
 * <FieldRow>
 *   <NodeTextField field={name} label="Name" />
 *   <NodeTextField field={path} label="Path" />
 * </FieldRow>
 *
 * <FieldRow cols={3}>
 *   <NodeNumberField field={width} label="Width" />
 *   <NodeNumberField field={height} label="Height" />
 *   <NodeNumberField field={depth} label="Depth" />
 * </FieldRow>
 * ```
 */
export const FieldRow: FC<FieldRowProps> = ({ children, cols = 2, className }) => (
  <div className={`grid ${colsClasses[cols]} gap-2 ${className ?? ''}`}>{children}</div>
);

export default FieldRow;
