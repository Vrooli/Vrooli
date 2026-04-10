// DOC: docs/reference/api-endpoints.md#scenario-documentation-tree
import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, ChevronDown, ChevronRight, FileText, Folder, RefreshCw } from "lucide-react";
import { Button } from "../../../shared/ui/button";
import { selectors } from "../../../consts/selectors";
import type { DocTreeNode } from "../../../shared/services/documentationApi";
import type { HealthTone } from "../../../shared/controllers/documentationController";

const severityClasses: Record<string, string> = {
  error: "ko-warning-pill-error",
  warning: "ko-warning-pill-warning",
  info: "ko-warning-pill-info",
};

const toneClasses: Record<HealthTone, string> = {
  good: "ko-tone-good",
  medium: "ko-tone-medium",
  poor: "ko-tone-poor",
};

export type DocTreeProps = {
  tree?: DocTreeNode;
  selectedPath: string | null;
  onSelectPath: (path: string) => void;
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string;
  onRefresh: () => void;
  /** Health score label to display in header (e.g., "85%") */
  healthScoreLabel?: string;
  /** Health tone for styling the badge */
  healthTone?: HealthTone;
  /** Called when health badge is clicked */
  onHealthClick?: () => void;
};

export function DocTree({
  tree,
  selectedPath,
  onSelectPath,
  isLoading,
  hasError,
  errorMessage,
  onRefresh,
  healthScoreLabel,
  healthTone,
  onHealthClick,
}: DocTreeProps) {
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (tree?.path) {
      setExpanded((prev) => ({ ...prev, [tree.path]: true }));
    }
  }, [tree?.path]);

  const nodes = useMemo(() => (tree ? [tree] : []), [tree]);

  const toggle = (path: string) => {
    setExpanded((prev) => ({ ...prev, [path]: !prev[path] }));
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-10">
        <RefreshCw className="h-5 w-5 ko-icon animate-spin" />
        <span className="ml-3 ko-text-sm ko-muted">Loading docs tree...</span>
      </div>
    );
  }

  if (hasError) {
    return (
      <div className="ko-alert ko-alert-danger">
        <div className="flex-1">
          <p className="ko-alert-title ko-text-danger-strong">Failed to load doc tree</p>
          <p className="ko-text-sm ko-text-danger-muted mt-1">{errorMessage}</p>
          <Button onClick={onRefresh} variant="danger" className="mt-3">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!tree) {
    return <div className="ko-panel p-4 ko-text-sm ko-muted">Select a scenario to view documentation.</div>;
  }

  const toneClass = healthTone ? toneClasses[healthTone] : "";

  return (
    <div className="ko-stack-sm" data-testid={selectors.explorer.docTree}>
      <div className="ko-doctree-header">
        <div className="ko-doctree-header-info">
          <span className="ko-text-xs ko-subtle">Root: {tree.path}</span>
        </div>
        <div className="ko-doctree-header-actions">
          {healthScoreLabel && onHealthClick && (
            <button
              type="button"
              className={`ko-health-badge-button ${toneClass}`}
              onClick={onHealthClick}
              aria-label="View documentation health details"
            >
              Health {healthScoreLabel}
            </button>
          )}
          <Button onClick={onRefresh} variant="outline" size="compact">
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="ko-tree">
        {nodes.map((node) => (
          <DocTreeNodeRow
            key={node.path}
            node={node}
            level={0}
            expanded={expanded}
            onToggle={toggle}
            onSelect={onSelectPath}
            selectedPath={selectedPath}
          />
        ))}
      </div>
    </div>
  );
}

type DocTreeNodeRowProps = {
  node: DocTreeNode;
  level: number;
  expanded: Record<string, boolean>;
  onToggle: (path: string) => void;
  onSelect: (path: string) => void;
  selectedPath: string | null;
};

function DocTreeNodeRow({ node, level, expanded, onToggle, onSelect, selectedPath }: DocTreeNodeRowProps) {
  const isDirectory = node.type === "directory";
  const isExpanded = expanded[node.path] ?? level === 0;
  const hasChildren = (node.children?.length ?? 0) > 0;
  const warningClass = node.warning?.severity ? severityClasses[node.warning.severity] : "";

  const handleClick = () => {
    if (isDirectory) {
      if (hasChildren) {
        onToggle(node.path);
      }
      return;
    }
    onSelect(node.path);
  };

  return (
    <div>
      <button
        type="button"
        onClick={handleClick}
        className={[
          "ko-tree-row",
          isDirectory ? "ko-tree-row-directory" : "ko-tree-row-file",
          selectedPath === node.path ? "ko-tree-row-active" : "",
        ].join(" ")}
        style={{ paddingLeft: `${level * 16 + 8}px` }}
      >
        <span className="ko-tree-icon">
          {isDirectory ? (
            hasChildren ? (
              isExpanded ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )
            ) : (
              <span className="h-4 w-4 inline-block" />
            )
          ) : null}
        </span>
        {isDirectory ? <Folder className="h-4 w-4 ko-icon" /> : <FileText className="h-4 w-4 ko-icon" />}
        <span className="flex-1 text-left">
          {node.name}
          {node.doc_type ? <span className="ko-pill ko-pill-muted ml-2">{node.doc_type}</span> : null}
        </span>
        {node.warning ? (
          <span className={`ko-warning-pill ${warningClass}`} title={node.warning.message}>
            <AlertTriangle className="h-3 w-3" />
            {node.warning.type}
          </span>
        ) : null}
      </button>

      {isDirectory && isExpanded && hasChildren
        ? node.children?.map((child) => (
            <DocTreeNodeRow
              key={child.path}
              node={child}
              level={level + 1}
              expanded={expanded}
              onToggle={onToggle}
              onSelect={onSelect}
              selectedPath={selectedPath}
            />
          ))
        : null}
    </div>
  );
}
