import { FileText, X } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";

export interface ViewingFileBlame {
  path: string;
  filename: string;
}

interface BlameModeHeaderProps {
  file: ViewingFileBlame;
  onExit: () => void;
  compact?: boolean;
}

export function BlameModeHeader({ file, onExit, compact }: BlameModeHeaderProps) {
  if (compact) {
    return (
      <header
        className="flex items-center justify-between px-3 py-2 border-b border-blue-800/50 bg-blue-950/30 backdrop-blur-sm pt-safe"
        data-testid="mobile-header"
      >
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <FileText className="h-4 w-4 text-blue-400 flex-shrink-0" />
          <Badge variant="info" className="flex-shrink-0 text-xs">
            File History
          </Badge>
          <span className="text-xs text-blue-200 truncate" title={file.path}>
            {file.filename}
          </span>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={onExit}
          className="gap-1 border-blue-600/50 text-blue-200 hover:bg-blue-900/30 text-xs px-2"
        >
          <X className="h-3.5 w-3.5" />
          Exit
        </Button>
      </header>
    );
  }

  return (
    <header
      className="relative z-30 flex items-center justify-between px-4 py-3 border-b border-blue-800/50 bg-blue-950/30 backdrop-blur-sm"
      data-testid="status-header"
    >
      <div className="flex items-center gap-4 min-w-0 flex-1">
        <div className="flex items-center gap-2 flex-shrink-0">
          <FileText className="h-4 w-4 text-blue-400" />
          <Badge variant="info" className="text-xs">
            File History
          </Badge>
        </div>

        <div className="flex items-center gap-2 min-w-0" data-testid="blame-file-info">
          <span className="text-sm text-blue-200 truncate" title={file.path}>
            {file.filename}
          </span>
          <span className="text-xs text-slate-500 truncate hidden sm:block" title={file.path}>
            ({file.path})
          </span>
        </div>
      </div>

      <div className="flex items-center gap-3 flex-shrink-0">
        <Button
          variant="outline"
          size="sm"
          onClick={onExit}
          className="gap-1.5 border-blue-600/50 text-blue-200 hover:bg-blue-900/30"
          data-testid="exit-blame-mode"
        >
          <X className="h-3.5 w-3.5" />
          Back to Working Directory
        </Button>
      </div>
    </header>
  );
}
