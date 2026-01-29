import { useEffect, useMemo, useRef, useState } from "react";
import { ArrowLeft, RefreshCw, Settings } from "lucide-react";
import { Button } from "../../../shared/ui/button";
import type { DocViewMode } from "../../../shared/hooks/viewerHooks";
import { CodeView } from "../../viewer/components/CodeView";
import { PreviewView } from "../../viewer/components/PreviewView";
import { ViewModeToggle } from "../../viewer/components/ViewModeToggle";
import { ResizeHandle } from "../../../shared/components/ResizeHandle";
import { selectors } from "../../../consts/selectors";

export type DocViewerProps = {
  /** The document path being viewed */
  path: string;
  /** Document content */
  content?: string;
  /** Loading state */
  isLoading: boolean;
  /** Error state */
  hasError: boolean;
  /** Error message to display */
  errorMessage: string;
  /** Called to refresh the document */
  onRefresh: () => void;
  /** Called when the back button is clicked */
  onBack: () => void;
  /** Current view mode */
  viewMode: DocViewMode;
  /** Called when view mode changes */
  onViewModeChange: (mode: DocViewMode) => void;
  /** Split ratio for code/preview (0.25-0.75) */
  splitRatio?: number;
  /** Called when split ratio changes during drag */
  onSplitResize?: (deltaX: number, containerWidth: number) => void;
  /** Whether to show the back button */
  showBackButton?: boolean;
  /** Document metadata for the dropdown */
  meta?: {
    path: string;
    docTypeLabel: string;
    sizeLabel: string;
    modifiedLabel: string;
  } | null;
};

export function DocViewer({
  path,
  content,
  isLoading,
  hasError,
  errorMessage,
  onRefresh,
  onBack,
  viewMode,
  onViewModeChange,
  splitRatio = 0.5,
  onSplitResize,
  showBackButton = true,
  meta,
}: DocViewerProps) {
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    if (!isDropdownOpen) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isDropdownOpen]);

  const fileName = useMemo(() => {
    const parts = path.split("/");
    return parts[parts.length - 1] ?? path;
  }, [path]);

  const handleSplitResize = (deltaX: number) => {
    if (onSplitResize && containerRef.current) {
      const containerWidth = containerRef.current.offsetWidth;
      onSplitResize(deltaX, containerWidth);
    }
  };

  return (
    <div className="ko-inline-viewer">
      {/* Header */}
      <div className="ko-inline-viewer-header">
        <div className="ko-inline-viewer-title">
          {showBackButton && (
            <Button
              type="button"
              variant="outline"
              size="compact"
              onClick={onBack}
              className="ko-mobile-back"
              aria-label="Back to tree"
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
          )}
          <span className="ko-inline-viewer-title-text" title={path}>
            {fileName}
          </span>
        </div>

        <div className="ko-inline-viewer-actions">
          <ViewModeToggle mode={viewMode} onChange={onViewModeChange} />
          <Button
            type="button"
            variant="outline"
            size="compact"
            onClick={onRefresh}
            disabled={isLoading}
            aria-label="Refresh document"
          >
            <RefreshCw className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`} />
          </Button>

          {/* Settings dropdown */}
          <div className="ko-viewer-dropdown" ref={dropdownRef}>
            <Button
              type="button"
              variant="outline"
              size="compact"
              onClick={() => setIsDropdownOpen(!isDropdownOpen)}
              aria-label="Document settings"
            >
              <Settings className="h-4 w-4" />
            </Button>
            {isDropdownOpen && (
              <div className="ko-viewer-dropdown-content">
                <div className="ko-stack-sm">
                  <p className="ko-text-sm font-semibold ko-text-strong">Document Info</p>
                  {meta ? (
                    <div className="ko-stack-xs">
                      <div>
                        <p className="ko-meta">Path</p>
                        <p className="ko-text-sm break-all">{meta.path}</p>
                      </div>
                      <div className="flex gap-4">
                        <div>
                          <p className="ko-meta">Doc type</p>
                          <p className="ko-text-sm">{meta.docTypeLabel}</p>
                        </div>
                        <div>
                          <p className="ko-meta">Size</p>
                          <p className="ko-text-sm">{meta.sizeLabel}</p>
                        </div>
                      </div>
                      <div>
                        <p className="ko-meta">Last modified</p>
                        <p className="ko-text-sm">{meta.modifiedLabel}</p>
                      </div>
                    </div>
                  ) : (
                    <p className="ko-text-sm ko-subtle">No metadata available.</p>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="ko-inline-viewer-content" ref={containerRef}>
        {viewMode === "split" ? (
          <div className="ko-viewer-split-container">
            <div
              className="ko-viewer-split-pane"
              style={{ width: `${splitRatio * 100}%` }}
            >
              <div className="ko-viewer-split-pane-scroll">
                <CodeView
                  content={content}
                  path={path}
                  isLoading={isLoading}
                  hasError={hasError}
                  errorMessage={errorMessage}
                />
              </div>
            </div>
            <ResizeHandle direction="vertical" onResize={handleSplitResize} />
            <div
              className="ko-viewer-split-pane"
              style={{ width: `${(1 - splitRatio) * 100}%` }}
            >
              <div className="ko-viewer-split-pane-scroll">
                <PreviewView
                  content={content}
                  isLoading={isLoading}
                  hasError={hasError}
                  errorMessage={errorMessage}
                />
              </div>
            </div>
          </div>
        ) : (
          <div className="ko-scroll-container" data-testid={selectors.viewer.codeView}>
            {viewMode === "code" ? (
              <CodeView
                content={content}
                path={path}
                isLoading={isLoading}
                hasError={hasError}
                errorMessage={errorMessage}
              />
            ) : (
              <PreviewView
                content={content}
                isLoading={isLoading}
                hasError={hasError}
                errorMessage={errorMessage}
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
}
