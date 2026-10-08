import { useEffect, useMemo, useRef, useState } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { MinusCircle } from "lucide-react";
import {
  ProposalState,
  type Proposal,
} from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { useEditProposal } from "../../lib/hooks-proposals";
import { describeFlag, hasPriorWork, kindMarker } from "./model";

export function ProposalFiles({
  proposal,
  selectedPath,
  onSelect,
  repoId,
  historical,
}: {
  proposal: Proposal;
  selectedPath?: string;
  onSelect: (path: string) => void;
  repoId: string | null;
  historical: boolean;
}) {
  const scroll = useRef<HTMLDivElement>(null);
  const edit = useEditProposal(repoId);
  const [error, setError] = useState<string | null>(null);
  const rows = useMemo(() => {
    const result: Array<{
      directory: string;
      file?: Proposal["files"][number];
    }> = [];
    let last = "";
    proposal.files.forEach((file) => {
      const directory = file.path.includes("/")
        ? file.path.slice(0, file.path.lastIndexOf("/"))
        : "Repository root";
      if (directory !== last) {
        result.push({ directory });
        last = directory;
      }
      result.push({ directory, file });
    });
    return result;
  }, [proposal.files]);
  const virtual = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scroll.current,
    estimateSize: (index) => (rows[index]?.file ? 64 : 34),
    overscan: 5,
    initialRect: { width: 300, height: 600 },
  });
  useEffect(() => {
    const index = rows.findIndex((row) => row.file?.path === selectedPath);
    if (index >= 0) virtual.scrollToIndex(index, { align: "auto" });
  }, [selectedPath, rows, virtual]);
  const exclude = async (path: string) => {
    setError(null);
    try {
      await edit.mutateAsync({
        id: proposal.id,
        expectedRevision: proposal.revision,
        removePaths: [path],
      });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    }
  };
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="shrink-0 border-b border-slate-800 px-4 py-3">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-slate-400">
          Proposed files · {proposal.files.length}
        </h2>
        <p className="mt-1 text-[11px] text-slate-500">
          Excluding a file leaves its code uncommitted.
        </p>
      </div>
      {error && (
        <p className="px-3 py-2 text-xs text-red-300" role="alert">
          {error}
        </p>
      )}
      <div
        ref={scroll}
        className="min-h-0 flex-1 overflow-auto"
        data-testid="proposal-files"
      >
        <div style={{ height: virtual.getTotalSize(), position: "relative" }}>
          {virtual.getVirtualItems().map((item) => {
            const row = rows[item.index];
            if (!row) return null;
            const file = row.file;
            const style = {
              position: "absolute" as const,
              top: 0,
              left: 0,
              width: "100%",
              height: item.size,
              transform: `translateY(${item.start}px)`,
            };
            if (!file)
              return (
                <div
                  key={`dir-${row.directory}`}
                  style={style}
                  className="truncate bg-slate-900/60 px-3 py-2 text-[11px] text-slate-500"
                  title={row.directory}
                >
                  {row.directory}
                </div>
              );
            const marker = kindMarker(file.kind);
            const uncertain = hasPriorWork(file);
            return (
              <div
                key={file.path}
                style={style}
                className={`flex items-center gap-1 border-b border-slate-800/60 px-2 ${selectedPath === file.path ? "bg-blue-950/40" : "hover:bg-slate-900/70"}`}
              >
                <span
                  className={`w-4 shrink-0 text-center font-mono text-xs ${marker.className}`}
                >
                  {marker.letter}
                </span>
                <button
                  className="min-h-11 min-w-0 flex-1 py-2 text-left"
                  onClick={() => onSelect(file.path)}
                  onKeyDown={(event) => {
                    const index = proposal.files.findIndex(
                      (value) => value.path === file.path,
                    );
                    const next =
                      event.key === "ArrowDown"
                        ? index + 1
                        : event.key === "ArrowUp"
                          ? index - 1
                          : event.key === "Home"
                            ? 0
                            : event.key === "End"
                              ? proposal.files.length - 1
                              : undefined;
                    if (next !== undefined) {
                      event.preventDefault();
                      const target = proposal.files[next];
                      if (target) {
                        onSelect(target.path);
                        const rowIndex = rows.findIndex(
                          (value) => value.file?.path === target.path,
                        );
                        if (rowIndex >= 0)
                          virtual.scrollToIndex(rowIndex, { align: "auto" });
                        requestAnimationFrame(() => {
                          const button = Array.from(
                            scroll.current?.querySelectorAll<HTMLButtonElement>(
                              "button[data-path]",
                            ) ?? [],
                          ).find((value) => value.dataset.path === target.path);
                          button?.focus();
                        });
                      }
                    }
                  }}
                  title={[
                    file.path,
                    ...file.flags.map((flag) => describeFlag(flag.code)),
                  ].join("\n")}
                  aria-pressed={selectedPath === file.path}
                  data-testid="proposal-file"
                  data-path={file.path}
                >
                  <span className="block truncate text-xs font-medium text-slate-200">
                    {file.path.split("/").pop()}
                  </span>
                  {uncertain && (
                    <span className="mt-1 block text-[10px] text-amber-300">
                      Includes earlier or unattributed work
                    </span>
                  )}
                </button>
                {proposal.state === ProposalState.OPEN && !historical && (
                  <button
                    className="inline-flex h-11 w-9 shrink-0 items-center justify-center rounded text-slate-500 hover:text-slate-200 disabled:opacity-30"
                    title={
                      proposal.files.length <= 1
                        ? "Close the proposal to leave its last file uncommitted"
                        : "Leave this file uncommitted"
                    }
                    aria-label={`Leave ${file.path.split("/").pop()} uncommitted`}
                    onClick={() => void exclude(file.path)}
                    disabled={edit.isPending || proposal.files.length <= 1}
                  >
                    <MinusCircle className="h-4 w-4" />
                  </button>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
