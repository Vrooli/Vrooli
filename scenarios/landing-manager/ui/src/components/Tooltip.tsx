import { useState } from 'react';

interface TooltipProps {
  children: React.ReactNode;
  content: string;
}

/**
 * Tooltip component for contextual help.
 * Shows tooltip on hover/focus with keyboard navigation support.
 */
export function Tooltip({ children, content }: TooltipProps) {
  const [show, setShow] = useState(false);

  return (
    <div className="relative inline-block">
      <button
        type="button"
        onMouseEnter={() => setShow(true)}
        onMouseLeave={() => setShow(false)}
        onFocus={() => setShow(true)}
        onBlur={() => setShow(false)}
        className="inline-flex items-center"
        aria-label={content}
      >
        {children}
      </button>
      {show && (
        <div
          role="tooltip"
          className="absolute z-10 left-1/2 -translate-x-1/2 bottom-full mb-2 px-3 py-2 text-xs text-slate-100 bg-slate-800 border border-slate-700 rounded-lg shadow-lg w-64 pointer-events-none"
        >
          {content}
          <div className="absolute left-1/2 -translate-x-1/2 top-full -mt-1 border-4 border-transparent border-t-slate-800"></div>
        </div>
      )}
    </div>
  );
}
