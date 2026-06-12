import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import Editor, { type OnChange } from '@monaco-editor/react';
import { Eye, PanelLeftOpen, PencilLine } from 'lucide-react';
import { useTheme } from '@/contexts/useTheme';
import { cn } from '@/lib/utils';
import { useResizableSplitPanel } from '@/hooks/useResizableSplitPanel';
import { useEditorPreferences } from '@/hooks/useEditorPreferences';
import { MarkdownRenderer } from '@/components/markdown';

export type MarkdownDisplayMode = 'raw' | 'preview';

interface MarkdownDisplayProps {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  placeholder?: string;
  className?: string;
  storageKey?: string;
  defaultMode?: MarkdownDisplayMode;
  splitBreakpoint?: number;
}

export function MarkdownDisplay({
  value,
  onChange,
  readOnly = false,
  placeholder = 'No content',
  className,
  storageKey = 'ecosystem-manager.markdownDisplay',
  defaultMode = 'raw',
  splitBreakpoint = 980,
}: MarkdownDisplayProps) {
  const { resolvedTheme } = useTheme();
  const { showLineNumbers } = useEditorPreferences();
  const layoutRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);
  const [mode, setMode] = useState<MarkdownDisplayMode>(() => {
    if (typeof window === 'undefined') return defaultMode;
    const stored = localStorage.getItem(`${storageKey}.mode`);
    return stored === 'preview' ? 'preview' : 'raw';
  });

  const {
    width: splitWidth,
    isResizing,
    isCollapsed,
    containerRef: splitContainerRef,
    handleResizeStart,
    expand,
    collapse,
  } = useResizableSplitPanel({
    storageKey: `${storageKey}.splitWidth`,
  });

  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(`${storageKey}.mode`, mode);
    }
  }, [mode, storageKey]);

  useEffect(() => {
    if (!layoutRef.current || typeof ResizeObserver === 'undefined') return;

    const handleResize = () => {
      if (!layoutRef.current) return;
      setContainerWidth(layoutRef.current.clientWidth);
    };

    handleResize();
    const observer = new ResizeObserver(handleResize);
    observer.observe(layoutRef.current);

    return () => observer.disconnect();
  }, []);

  const isSplitCapable = containerWidth >= splitBreakpoint;
  const showSplit = mode === 'preview' && isSplitCapable && !isCollapsed;

  const handleMonacoChange: OnChange = useCallback(
    (newValue) => {
      if (readOnly || !onChange || newValue === undefined) return;
      onChange(newValue);
    },
    [onChange, readOnly]
  );

  const editorPane = useMemo(
    () => (
      <div className="flex h-full flex-col">
        <div className="flex items-center justify-between border-b border-white/10 bg-slate-900/80 px-3 py-1.5">
          <div className="text-xs uppercase tracking-wide text-slate-400">Raw markdown</div>
          {readOnly && (
            <span className="rounded bg-white/5 px-2 py-0.5 text-[11px] text-slate-400">Read-only</span>
          )}
        </div>
        <div className="flex-1">
          <Editor
            height="100%"
            defaultLanguage="markdown"
            value={value}
            onChange={handleMonacoChange}
            theme={resolvedTheme === 'dark' ? 'vs-dark' : 'vs-light'}
            options={{
              readOnly,
              minimap: { enabled: false },
              wordWrap: 'on',
              lineNumbers: showLineNumbers ? 'on' : 'off',
              fontSize: 13,
              fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
              tabSize: 2,
              scrollBeyondLastLine: false,
              padding: { top: 12, bottom: 12 },
              renderLineHighlight: 'line',
              smoothScrolling: true,
              scrollbar: {
                vertical: 'auto',
                horizontal: 'auto',
                verticalScrollbarSize: 8,
                horizontalScrollbarSize: 8,
              },
              overviewRulerBorder: false,
              hideCursorInOverviewRuler: true,
              folding: true,
              foldingStrategy: 'indentation',
              automaticLayout: true,
            }}
          />
        </div>
      </div>
    ),
    [handleMonacoChange, readOnly, resolvedTheme, showLineNumbers, value]
  );

  const previewPane = useMemo(
    () => (
      <div className="flex h-full flex-col">
        <div className="flex items-center justify-between border-b border-white/10 bg-slate-900/80 px-3 py-1.5">
          <div className="text-xs uppercase tracking-wide text-slate-400">Preview</div>
        </div>
        <div className="flex-1 overflow-y-auto p-4">
          {value ? (
            <MarkdownRenderer content={value} />
          ) : (
            <div className="text-sm text-slate-500">{placeholder}</div>
          )}
        </div>
      </div>
    ),
    [placeholder, value]
  );

  return (
    <div ref={layoutRef} className={cn('flex h-full min-h-0 flex-col overflow-hidden rounded-md border border-white/10 bg-slate-950/60', className)}>
      <div className="flex items-center justify-between border-b border-white/10 bg-slate-900/70 px-3 py-2">
        <div className="text-xs text-slate-400">
          {mode === 'raw' ? 'Raw mode' : showSplit ? 'Split preview mode' : 'Preview mode'}
        </div>
        <div className="flex items-center gap-2">
          {mode === 'preview' && isCollapsed && isSplitCapable && (
            <button
              type="button"
              onClick={expand}
              className="rounded p-1.5 text-slate-300 transition-colors hover:bg-white/10 hover:text-slate-100"
              title="Expand raw editor"
              aria-label="Expand raw editor"
            >
              <PanelLeftOpen className="h-4 w-4" />
            </button>
          )}
          {mode === 'preview' && showSplit && (
            <button
              type="button"
              onClick={collapse}
              className="rounded px-2 py-1 text-xs text-slate-300 transition-colors hover:bg-white/10 hover:text-slate-100"
              title="Hide raw editor"
            >
              Hide raw
            </button>
          )}
          <button
            type="button"
            onClick={() => setMode('raw')}
            className={cn(
              'inline-flex items-center gap-1.5 rounded px-2 py-1 text-xs transition-colors',
              mode === 'raw' ? 'bg-blue-500/20 text-blue-200' : 'text-slate-300 hover:bg-white/10 hover:text-slate-100'
            )}
            aria-pressed={mode === 'raw'}
          >
            <PencilLine className="h-3.5 w-3.5" />
            Raw
          </button>
          <button
            type="button"
            onClick={() => setMode('preview')}
            className={cn(
              'inline-flex items-center gap-1.5 rounded px-2 py-1 text-xs transition-colors',
              mode === 'preview' ? 'bg-blue-500/20 text-blue-200' : 'text-slate-300 hover:bg-white/10 hover:text-slate-100'
            )}
            aria-pressed={mode === 'preview'}
          >
            <Eye className="h-3.5 w-3.5" />
            Preview
          </button>
        </div>
      </div>

      <div className="min-h-0 flex-1">
        {mode === 'raw' ? (
          editorPane
        ) : showSplit ? (
          <div
            ref={splitContainerRef}
            className={cn('flex h-full min-h-0', isResizing && 'select-none')}
          >
            <div className="h-full flex-shrink-0" style={{ width: splitWidth }}>
              {editorPane}
            </div>
            <div
              role="separator"
              aria-orientation="vertical"
              aria-label="Resize markdown split"
              tabIndex={0}
              onMouseDown={handleResizeStart}
              className="group relative w-3 flex-shrink-0 cursor-col-resize"
            >
              <div className="absolute right-1 top-0 h-full w-0.5 bg-white/20 transition-colors group-hover:bg-blue-300/70" />
            </div>
            <div className="min-w-0 flex-1">{previewPane}</div>
          </div>
        ) : (
          previewPane
        )}
      </div>
    </div>
  );
}
