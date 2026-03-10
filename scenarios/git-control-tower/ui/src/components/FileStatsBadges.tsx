import { Badge } from "./ui/badge";

interface FileStatsBadgesProps {
  staged: number;
  unstaged: number;
  untracked: number;
  conflicts: number;
  cleanDetails?: string;
}

export function FileStatsBadges({
  staged,
  unstaged,
  untracked,
  conflicts,
  cleanDetails
}: FileStatsBadgesProps) {
  const isClean = staged === 0 && unstaged === 0 && untracked === 0 && conflicts === 0;

  if (isClean) {
    return (
      <div className="flex items-center gap-3" data-testid="file-stats">
        <span className="text-xs text-slate-500">
          {cleanDetails ? `Working tree clean (${cleanDetails})` : "Working tree clean"}
        </span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-3" data-testid="file-stats">
      {staged > 0 && (
        <Badge variant="staged">{staged} staged</Badge>
      )}
      {unstaged > 0 && (
        <Badge variant="unstaged">{unstaged} modified</Badge>
      )}
      {untracked > 0 && (
        <Badge variant="untracked">{untracked} untracked</Badge>
      )}
      {conflicts > 0 && (
        <Badge variant="conflict">{conflicts} conflicts</Badge>
      )}
    </div>
  );
}
