import { FC } from 'react';
import { Globe } from 'lucide-react';
import { useUrlInheritance } from '@hooks/useUrlInheritance';

export interface NodeUrlFieldProps {
  /** The node ID to get URL inheritance for */
  nodeId: string;
  /** Label text (default: "Page URL") */
  label?: string;
  /** Error message when no URL is available (default: "Provide a URL to target this node.") */
  errorMessage?: string;
  /** Additional className for the wrapper div */
  className?: string;
}

/**
 * URL input field with inheritance from upstream nodes.
 *
 * Features:
 * - Inherits URL from upstream NavigateNode if not set
 * - Shows "Inherits {url}" message when using inherited URL
 * - Reset button to clear custom URL and use inherited
 * - Error message when no URL is available
 *
 * @example
 * ```tsx
 * // Simple usage - just pass nodeId
 * <NodeUrlField nodeId={id} />
 *
 * // With custom error message
 * <NodeUrlField
 *   nodeId={id}
 *   errorMessage="Provide a URL to target this click."
 * />
 * ```
 */
export const NodeUrlField: FC<NodeUrlFieldProps> = ({
  nodeId,
  label = 'Page URL',
  errorMessage = 'Provide a URL to target this node.',
  className,
}) => {
  const {
    urlDraft,
    setUrlDraft,
    effectiveUrl,
    hasCustomUrl,
    upstreamUrl,
    commitUrl,
    resetUrl,
    handleUrlKeyDown,
  } = useUrlInheritance(nodeId);

  return (
    <div className={className ?? 'mb-2'}>
      <label className="flex items-center gap-1 text-[11px] font-semibold uppercase tracking-wide text-gray-400">
        <Globe size={12} className="text-blue-400" />
        {label}
      </label>
      <div className="mt-1 flex items-center gap-2">
        <input
          type="text"
          placeholder={upstreamUrl ?? 'https://example.com'}
          className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          value={urlDraft}
          onChange={(event) => setUrlDraft(event.target.value)}
          onBlur={commitUrl}
          onKeyDown={handleUrlKeyDown}
        />
        {hasCustomUrl && (
          <button
            type="button"
            className="px-2 py-1 text-[11px] rounded border border-gray-700 text-gray-300 hover:bg-gray-700 transition-colors"
            onClick={resetUrl}
          >
            Reset
          </button>
        )}
      </div>
      {!hasCustomUrl && upstreamUrl && (
        <p className="mt-1 text-[10px] text-gray-500 truncate" title={upstreamUrl}>
          Inherits {upstreamUrl}
        </p>
      )}
      {!effectiveUrl && !upstreamUrl && (
        <p className="mt-1 text-[10px] text-red-400">{errorMessage}</p>
      )}
    </div>
  );
};

export default NodeUrlField;
