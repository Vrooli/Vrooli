import { Loader2, ExternalLink } from "lucide-react";
import { useCaptureContent } from "../../hooks/useCaptureContent";
import { selectors } from "../../consts/selectors";

interface CaptureContentViewerProps {
  backlogKind: string;
  backlogName: string;
  capturePath: string;
  loadingLabel?: string;
  openLabel?: string;
  testId?: string;
  renderContent: (content: string) => React.ReactNode;
}

export function CaptureContentViewer({
  backlogKind,
  backlogName,
  capturePath,
  loadingLabel = "Loading...",
  openLabel = "Open full content",
  testId,
  renderContent,
}: CaptureContentViewerProps) {
  const { content, isLoading, error, isTruncated, captureUrl } = useCaptureContent(
    backlogKind,
    backlogName,
    capturePath,
  );

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 rounded bg-slate-900 p-3 text-xs text-slate-400 dark:bg-slate-950">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        {loadingLabel}
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded bg-slate-900 p-3 text-xs text-red-400 dark:bg-slate-950">
        {error}
      </div>
    );
  }

  return (
    <div data-testid={testId}>
      {renderContent(content ?? "")}
      {isTruncated && (
        <a
          href={captureUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-1 inline-flex items-center gap-1 text-xs text-violet-400 hover:text-violet-300"
          data-testid={selectors.evidence.truncatedLink}
        >
          <ExternalLink className="h-3 w-3" />
          {openLabel}
        </a>
      )}
    </div>
  );
}
