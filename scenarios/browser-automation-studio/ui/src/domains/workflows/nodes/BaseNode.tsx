/**
 * BaseNode Component
 *
 * Provides the common structural wrapper for workflow nodes:
 * - Outer wrapper with selection state
 * - Top and bottom handles
 * - Header with icon and title
 *
 * This eliminates ~15 lines of boilerplate per node component.
 */

import { memo, ReactNode, FC } from 'react';
import { Handle, Position } from 'reactflow';
import type { LucideIcon } from 'lucide-react';

export interface BaseNodeProps {
  /** Whether the node is selected */
  selected: boolean;
  /** The icon to display in the header */
  icon: LucideIcon;
  /** CSS classes for the icon (e.g., "text-yellow-400") */
  iconClassName?: string;
  /** The title to display in the header */
  title: string;
  /** The node content */
  children: ReactNode;
  /** Whether to show the top (target) handle - defaults to true */
  showTopHandle?: boolean;
  /** Whether to show the bottom (source) handle - defaults to true */
  showBottomHandle?: boolean;
  /** Additional CSS classes for the wrapper */
  className?: string;
}

/**
 * Base wrapper component for workflow nodes.
 *
 * @example
 * ```tsx
 * <BaseNode
 *   selected={selected}
 *   icon={MousePointer}
 *   iconClassName="text-green-400"
 *   title="Click"
 * >
 *   <input ... />
 *   <ResiliencePanel ... />
 * </BaseNode>
 * ```
 */
const BaseNode: FC<BaseNodeProps> = ({
  selected,
  icon: Icon,
  iconClassName = 'text-gray-400',
  title,
  children,
  showTopHandle = true,
  showBottomHandle = true,
  className = '',
}) => {
  return (
    <div className={`workflow-node ${selected ? 'selected' : ''} ${className}`.trim()}>
      {showTopHandle && (
        <Handle type="target" position={Position.Top} className="node-handle" />
      )}

      <div className="flex items-center gap-2 mb-2">
        <Icon size={16} className={iconClassName} />
        <span className="font-semibold text-sm">{title}</span>
      </div>

      {children}

      {showBottomHandle && (
        <Handle type="source" position={Position.Bottom} className="node-handle" />
      )}
    </div>
  );
};

export default memo(BaseNode);
