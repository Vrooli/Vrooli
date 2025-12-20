/**
 * DOMHierarchyNav Component
 *
 * Displays the DOM context for a selected element and allows
 * navigation to parent/child elements in the hierarchy.
 */

import { FC, memo } from 'react';
import { ArrowUp, ArrowDown } from 'lucide-react';
import type { ElementHierarchyEntry } from '@/types/elements';
import { summarizeElement, stringifyPath } from '../utils/elementHierarchy';

export interface DOMHierarchyNavProps {
  /** The list of hierarchy candidates (from child to parent) */
  candidates: ElementHierarchyEntry[];
  /** Currently selected index in the candidates array */
  selectedIndex: number;
  /** Callback when user wants to shift selection (delta: +1 for parent, -1 for child) */
  onShift: (delta: number) => void;
}

const DOMHierarchyNav: FC<DOMHierarchyNavProps> = ({ candidates, selectedIndex, onShift }) => {
  const hasSelection = candidates.length > 0 && selectedIndex >= 0 && selectedIndex < candidates.length;

  if (!hasSelection) {
    return null;
  }

  const selectedEntry = candidates[selectedIndex];
  const canSelectParent = selectedIndex < candidates.length - 1;
  const canSelectChild = selectedIndex > 0;

  return (
    <div className="mb-3 rounded-md border border-gray-800 bg-flow-bg/70 px-2 py-2">
      <div className="flex items-center justify-between gap-2">
        <span className="text-[11px] font-semibold uppercase tracking-wide text-gray-400">
          DOM context
        </span>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => onShift(1)}
            disabled={!canSelectParent}
            className={`inline-flex items-center gap-1 rounded border px-2 py-1 text-[11px] transition-colors ${
              canSelectParent
                ? 'border-gray-700 text-gray-300 hover:border-flow-accent hover:text-white'
                : 'border-gray-800 text-gray-600 cursor-not-allowed'
            }`}
            title="Select parent element"
          >
            <ArrowUp size={12} />
            Parent
          </button>
          <button
            type="button"
            onClick={() => onShift(-1)}
            disabled={!canSelectChild}
            className={`inline-flex items-center gap-1 rounded border px-2 py-1 text-[11px] transition-colors ${
              canSelectChild
                ? 'border-gray-700 text-gray-300 hover:border-flow-accent hover:text-white'
                : 'border-gray-800 text-gray-600 cursor-not-allowed'
            }`}
            title="Select child element"
          >
            <ArrowDown size={12} />
            Child
          </button>
        </div>
      </div>
      <div className="mt-1 text-[10px] text-gray-500">
        Depth {selectedIndex + 1} / {candidates.length}
      </div>
      <div
        className="mt-1 text-[12px] text-gray-200 truncate"
        title={summarizeElement(selectedEntry)}
      >
        {summarizeElement(selectedEntry)}
      </div>
      <div
        className="mt-1 text-[10px] font-mono text-gray-500 break-words"
        title={stringifyPath(selectedEntry)}
      >
        {stringifyPath(selectedEntry)}
      </div>
    </div>
  );
};

export default memo(DOMHierarchyNav);
