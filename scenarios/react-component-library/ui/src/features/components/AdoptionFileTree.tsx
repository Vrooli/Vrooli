import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { FileCode2, Folder } from "lucide-react";

import { adoptionsClient, type ResolvedVersionFile } from "../../api/adoptions";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { useTranslation } from "../../i18n";
import type { TemplateOption } from "./adoptionTemplates";

/** FlatFile is the version's file set used for the no-manifest fallback row. */
export interface FlatFile {
  path: string;
  isEntry: boolean;
}

interface AdoptionFileTreeProps {
  componentId: string;
  /** Version to resolve placement for; empty resolves the latest. */
  version?: string;
  /** The version's flat file set, used for the fallback tab row. */
  files: FlatFile[];
  /** Currently open file (library basename); "" means the entry file. */
  selectedFile: string;
  /** Called with the library basename to open ("" for the entry file). */
  onSelectFile: (libraryPath: string) => void;
  /** Selected template id (placement authority). */
  template: string;
  /** Available templates for the seam. */
  templates: readonly TemplateOption[];
  /** Called when the operator picks a different template. */
  onSelectTemplate: (template: string) => void;
}

interface TreeNode {
  name: string;
  children: Map<string, TreeNode>;
  file?: ResolvedVersionFile;
}

function buildTree(files: ResolvedVersionFile[]): TreeNode {
  const root: TreeNode = { name: "", children: new Map() };
  for (const file of files) {
    const parts = file.targetPath.split("/").filter(Boolean);
    let node = root;
    parts.forEach((part: string, index: number) => {
      let child = node.children.get(part);
      if (!child) {
        child = { name: part, children: new Map() };
        node.children.set(part, child);
      }
      if (index === parts.length - 1) child.file = file;
      node = child;
    });
  }
  return root;
}

// The library basename the editor opens for a resolved leaf. Entry files map to
// "" so the editor loads the component's current renderable artifact.
function fileKey(file: ResolvedVersionFile): string {
  return file.isEntry ? "" : file.libraryPath;
}

function TreeRows({
  node,
  depth,
  selectedFile,
  onSelectFile,
}: {
  node: TreeNode;
  depth: number;
  selectedFile: string;
  onSelectFile: (libraryPath: string) => void;
}) {
  const { t } = useTranslation();
  const entries = [...node.children.values()].sort((a, b) => {
    // Directories first, then files, each alphabetical.
    const aDir = a.file ? 1 : 0;
    const bDir = b.file ? 1 : 0;
    if (aDir !== bDir) return aDir - bDir;
    return a.name.localeCompare(b.name);
  });
  return (
    <>
      {entries.map((child) => {
        const indent = { paddingLeft: `${depth * 12 + 8}px` };
        if (!child.file) {
          return (
            <li key={child.name}>
              <div
                className="flex items-center gap-1.5 py-1 text-xs text-app-muted-foreground"
                style={indent}
              >
                <Folder aria-hidden className="h-3.5 w-3.5 shrink-0" />
                <span className="truncate">{child.name}</span>
              </div>
              <ul>
                <TreeRows
                  node={child}
                  depth={depth + 1}
                  selectedFile={selectedFile}
                  onSelectFile={onSelectFile}
                />
              </ul>
            </li>
          );
        }
        const file = child.file;
        const active = fileKey(file) === selectedFile;
        return (
          <li key={child.name}>
            <button
              type="button"
              data-testid={selectors.components.editor.fileTreeNode}
              data-slot={file.slot}
              data-entry={file.isEntry ? "true" : "false"}
              aria-current={active ? "true" : undefined}
              onClick={() => onSelectFile(fileKey(file))}
              className={`flex w-full items-center gap-1.5 rounded-control py-1 pr-2 text-left text-xs transition-colors ${
                active
                  ? "bg-app-primary/10 text-app-foreground"
                  : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
              }`}
              style={indent}
            >
              <FileCode2 aria-hidden className="h-3.5 w-3.5 shrink-0" />
              <span className="truncate font-mono">{child.name}</span>
              {file.slot && (
                <span className="ml-auto shrink-0 rounded bg-app-surface-muted px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-app-muted-foreground">
                  {file.slot}
                </span>
              )}
              {file.isEntry && (
                <span className="shrink-0 text-[10px] uppercase tracking-wide text-app-primary">
                  {t("components.editor.entryTag", { defaultValue: "entry" })}
                </span>
              )}
            </button>
          </li>
        );
      })}
    </>
  );
}

/**
 * AdoptionFileTree renders where each file of a multi-file component version
 * lands in an adopting scenario, using the selected template's UI manifest as
 * placement authority. Each leaf opens that file in the editor. When no
 * manifest resolves (or the version is single-file) it falls back to the flat
 * file-tab row so the panel never loses the ability to switch files.
 */
export function AdoptionFileTree({
  componentId,
  version,
  files,
  selectedFile,
  onSelectFile,
  template,
  templates,
  onSelectTemplate,
}: AdoptionFileTreeProps) {
  const { t } = useTranslation();
  const placementQuery = useQuery({
    queryKey: ["adoptions", "placement", componentId, version ?? "latest", template],
    queryFn: () =>
      adoptionsClient.resolveAdoptionPath({
        componentId,
        template,
        ...(version ? { version } : {}),
      }),
    enabled: Boolean(componentId),
  });

  const resolved = placementQuery.data;
  const tree = useMemo(
    () => (resolved?.files.length ? buildTree(resolved.files) : null),
    [resolved],
  );
  const manifestResolved = Boolean(resolved?.manifestResolved && tree);
  const multiTemplate = templates.length > 1;

  const templateSeam = (
    <div className="flex items-center gap-1.5 text-[11px] text-app-muted-foreground">
      <span>{t("components.editor.templateLabel", { defaultValue: "Template" })}</span>
      {multiTemplate ? (
        <select
          data-testid={selectors.components.editor.templateSelect}
          value={template}
          onChange={(event) => onSelectTemplate(event.target.value)}
          className="rounded-control border border-app-border bg-app-surface px-1.5 py-0.5 text-[11px] text-app-foreground"
        >
          {templates.map((option) => (
            <option key={option.id} value={option.id}>
              {option.label}
            </option>
          ))}
        </select>
      ) : (
        <span
          data-testid={selectors.components.editor.templateSelect}
          className="rounded-control bg-app-surface-muted px-1.5 py-0.5 font-mono text-app-foreground"
        >
          {templates[0]?.label ?? template}
        </span>
      )}
    </div>
  );

  // Fallback: no manifest placement (or the RPC hasn't resolved). Render the
  // flat file-tab row so switching files still works.
  if (!manifestResolved) {
    return (
      <div className="flex min-w-0 flex-col gap-1">
        <div
          className="flex min-w-0 gap-1 overflow-x-auto"
          data-testid={selectors.components.editor.fileTabs}
        >
          {files.map((file) => (
            <Button
              key={file.path}
              variant={
                selectedFile === file.path || (!selectedFile && file.isEntry)
                  ? "primary"
                  : "secondary"
              }
              className="h-7 shrink-0 px-2 text-xs"
              onClick={() => onSelectFile(file.isEntry ? "" : file.path)}
            >
              {file.path}
            </Button>
          ))}
        </div>
        {templateSeam}
      </div>
    );
  }

  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <div className="flex min-w-0 items-center justify-between gap-2">
        <span className="text-[11px] font-medium uppercase tracking-wide text-app-muted-foreground">
          {t("components.editor.placementHeading", { defaultValue: "Placement" })}
        </span>
        {templateSeam}
      </div>
      <ul
        data-testid={selectors.components.editor.fileTree}
        className="min-w-0 rounded-control bg-app-background py-1"
      >
        <TreeRows
          node={tree as TreeNode}
          depth={0}
          selectedFile={selectedFile}
          onSelectFile={onSelectFile}
        />
      </ul>
    </div>
  );
}
