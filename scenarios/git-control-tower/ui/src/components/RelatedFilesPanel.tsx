import { useMemo } from "react";
import { ChevronLeft, FileCode, TestTube, FileText, Folder, Link, Loader2, AlertCircle } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { useRelatedFiles } from "../lib/hooks";
import type { RelatedFile, RelationType } from "../lib/api";

interface RelatedFilesPanelProps {
  forPath: string;
  onBack: () => void;
  onSelectFile: (path: string) => void;
}

// Group related files by relation type
interface GroupedRelatedFiles {
  imports: RelatedFile[];
  importedBy: RelatedFile[];
  tests: RelatedFile[];
  index: RelatedFile[];
  types: RelatedFile[];
}

function groupRelatedFiles(files: RelatedFile[]): GroupedRelatedFiles {
  const grouped: GroupedRelatedFiles = {
    imports: [],
    importedBy: [],
    tests: [],
    index: [],
    types: []
  };

  for (const file of files) {
    switch (file.relation_type) {
      case "imports":
        grouped.imports.push(file);
        break;
      case "imported_by":
        grouped.importedBy.push(file);
        break;
      case "test":
        grouped.tests.push(file);
        break;
      case "index":
        grouped.index.push(file);
        break;
      case "types":
        grouped.types.push(file);
        break;
    }
  }

  return grouped;
}

// Get icon for relation type
function getRelationIcon(type: RelationType) {
  switch (type) {
    case "imports":
      return Link;
    case "imported_by":
      return Link;
    case "test":
      return TestTube;
    case "index":
      return Folder;
    case "types":
      return FileText;
    default:
      return FileCode;
  }
}

// Get color for relation type
function getRelationColor(type: RelationType): string {
  switch (type) {
    case "imports":
      return "text-blue-400";
    case "imported_by":
      return "text-purple-400";
    case "test":
      return "text-emerald-400";
    case "index":
      return "text-amber-400";
    case "types":
      return "text-cyan-400";
    default:
      return "text-slate-400";
  }
}

// Section header component
function SectionHeader({ title, count, icon: Icon, color }: {
  title: string;
  count: number;
  icon: typeof FileCode;
  color: string;
}) {
  return (
    <div className={`flex items-center gap-2 px-3 py-2 text-xs font-medium uppercase tracking-wide ${color} bg-slate-800/50`}>
      <Icon className="h-3.5 w-3.5" />
      <span>{title}</span>
      <span className="ml-auto text-slate-500">{count}</span>
    </div>
  );
}

// File item component
function FileItem({ file, onSelect }: { file: RelatedFile; onSelect: () => void }) {
  const Icon = getRelationIcon(file.relation_type);
  const color = getRelationColor(file.relation_type);

  return (
    <button
      type="button"
      onClick={onSelect}
      className="w-full flex items-center gap-3 px-3 py-2 text-left hover:bg-slate-800/50 transition-colors"
    >
      <Icon className={`h-4 w-4 flex-shrink-0 ${color}`} />
      <span className="flex-1 truncate font-mono text-xs text-slate-300">
        {file.path}
      </span>
    </button>
  );
}

export function RelatedFilesPanel({ forPath, onBack, onSelectFile }: RelatedFilesPanelProps) {
  const { data, isLoading, error } = useRelatedFiles(forPath);

  const grouped = useMemo(() => {
    if (!data?.related) return null;
    return groupRelatedFiles(data.related);
  }, [data?.related]);

  const hasAnyRelated = grouped && (
    grouped.imports.length > 0 ||
    grouped.importedBy.length > 0 ||
    grouped.tests.length > 0 ||
    grouped.index.length > 0 ||
    grouped.types.length > 0
  );

  return (
    <Card className="h-full flex flex-col" data-testid="related-files-panel">
      <CardHeader className="py-3 flex-row items-center gap-2 space-y-0">
        <button
          type="button"
          onClick={onBack}
          className="p-1 -ml-1 text-slate-400 hover:text-slate-200 transition-colors"
          aria-label="Back to changes"
          data-testid="related-files-back"
        >
          <ChevronLeft className="h-5 w-5" />
        </button>
        <CardTitle className="flex-1 min-w-0">
          <span className="text-xs text-slate-400">Related to</span>
          <p className="font-mono text-xs truncate text-slate-200 mt-0.5" title={forPath}>
            {forPath.split("/").pop()}
          </p>
        </CardTitle>
      </CardHeader>

      <CardContent className="flex-1 p-0 overflow-hidden">
        <ScrollArea className="h-full">
          {/* Loading state */}
          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-slate-500" />
            </div>
          )}

          {/* Error state */}
          {error && !isLoading && (
            <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
              <AlertCircle className="h-8 w-8 text-red-400 mb-3" />
              <p className="text-sm text-red-400">Failed to load related files</p>
              <p className="text-xs text-slate-500 mt-1">{error.message}</p>
            </div>
          )}

          {/* Empty state */}
          {!isLoading && !error && !hasAnyRelated && (
            <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
              <Link className="h-8 w-8 text-slate-600 mb-3" />
              <p className="text-sm text-slate-500">No related files found</p>
              <p className="text-xs text-slate-600 mt-1">
                This file doesn't import other files or have associated tests
              </p>
            </div>
          )}

          {/* Related files list */}
          {!isLoading && !error && grouped && hasAnyRelated && (
            <div className="divide-y divide-slate-800">
              {/* Imports section */}
              {grouped.imports.length > 0 && (
                <div>
                  <SectionHeader
                    title="Imports"
                    count={grouped.imports.length}
                    icon={Link}
                    color="text-blue-400"
                  />
                  <div>
                    {grouped.imports.map((file) => (
                      <FileItem
                        key={file.path}
                        file={file}
                        onSelect={() => onSelectFile(file.path)}
                      />
                    ))}
                  </div>
                </div>
              )}

              {/* Imported By section */}
              {grouped.importedBy.length > 0 && (
                <div>
                  <SectionHeader
                    title="Imported By"
                    count={grouped.importedBy.length}
                    icon={Link}
                    color="text-purple-400"
                  />
                  <div>
                    {grouped.importedBy.map((file) => (
                      <FileItem
                        key={file.path}
                        file={file}
                        onSelect={() => onSelectFile(file.path)}
                      />
                    ))}
                  </div>
                </div>
              )}

              {/* Tests section */}
              {grouped.tests.length > 0 && (
                <div>
                  <SectionHeader
                    title="Tests"
                    count={grouped.tests.length}
                    icon={TestTube}
                    color="text-emerald-400"
                  />
                  <div>
                    {grouped.tests.map((file) => (
                      <FileItem
                        key={file.path}
                        file={file}
                        onSelect={() => onSelectFile(file.path)}
                      />
                    ))}
                  </div>
                </div>
              )}

              {/* Index section */}
              {grouped.index.length > 0 && (
                <div>
                  <SectionHeader
                    title="Index"
                    count={grouped.index.length}
                    icon={Folder}
                    color="text-amber-400"
                  />
                  <div>
                    {grouped.index.map((file) => (
                      <FileItem
                        key={file.path}
                        file={file}
                        onSelect={() => onSelectFile(file.path)}
                      />
                    ))}
                  </div>
                </div>
              )}

              {/* Types section */}
              {grouped.types.length > 0 && (
                <div>
                  <SectionHeader
                    title="Types"
                    count={grouped.types.length}
                    icon={FileText}
                    color="text-cyan-400"
                  />
                  <div>
                    {grouped.types.map((file) => (
                      <FileItem
                        key={file.path}
                        file={file}
                        onSelect={() => onSelectFile(file.path)}
                      />
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
