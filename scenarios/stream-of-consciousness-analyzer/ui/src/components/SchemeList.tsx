// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2 } from "lucide-react";
import { listSchemes, createScheme, deleteScheme } from "../lib/api";
import { useMutationErrors } from "../hooks/useMutationErrors";
import { ErrorBanner } from "./ErrorBanner";
import type { Scheme } from "../lib/types";

interface Props {
  activeSchemeId: string | null;
  onSelect: (scheme: Scheme) => void;
}

export function SchemeList({ activeSchemeId, onSelect }: Props) {
  const qc = useQueryClient();
  const { data: schemes = [], isLoading, error: listError, refetch } = useQuery({ queryKey: ["schemes"], queryFn: listSchemes });

  const createMut = useMutation({
    mutationFn: () => createScheme("Untitled"),
    onSuccess: (scheme) => {
      qc.invalidateQueries({ queryKey: ["schemes"] });
      onSelect(scheme);
    },
  });

  const deleteMut = useMutation({
    mutationFn: deleteScheme,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["schemes"] }),
  });

  const { activeError: mutError, resetAll } = useMutationErrors([createMut, deleteMut]);
  const activeError = listError || mutError;

  return (
    <div data-testid="scheme-list" className="w-56 border-r border-white/10 bg-slate-900 flex flex-col h-full">
      <div className="flex items-center justify-between px-3 py-2 border-b border-white/10">
        <span className="text-xs uppercase tracking-widest text-slate-400">Schemes</span>
        <button
          data-testid="create-scheme-btn"
          onClick={() => createMut.mutate()}
          disabled={createMut.isPending}
          className="p-1 rounded hover:bg-white/10 text-slate-400 hover:text-white disabled:opacity-40"
          aria-label="New scheme"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>
      {activeError && (
        <div className="px-2 pt-2">
          <ErrorBanner
            error={activeError}
            onRetry={() => {
              if (listError) refetch();
              resetAll();
            }}
            onDismiss={resetAll}
          />
        </div>
      )}
      <div className="flex-1 overflow-y-auto">
        {isLoading && <p className="p-3 text-sm text-slate-500">Loading...</p>}
        {schemes.map((s) => (
          <div
            key={s.id}
            data-testid="scheme-item"
            role="button"
            tabIndex={0}
            onClick={() => onSelect(s)}
            onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") onSelect(s); }}
            aria-label={`Select scheme: ${s.name}`}
            aria-current={s.id === activeSchemeId ? "true" : undefined}
            className={`flex items-center justify-between px-3 py-2 cursor-pointer text-sm group ${
              s.id === activeSchemeId ? "bg-white/10 text-white" : "text-slate-300 hover:bg-white/5"
            }`}
          >
            <span className="truncate">{s.name}</span>
            <button
              onClick={(e) => {
                e.stopPropagation();
                deleteMut.mutate(s.id);
              }}
              disabled={deleteMut.isPending}
              className="p-1 rounded opacity-0 group-hover:opacity-100 hover:bg-white/10 text-slate-500 hover:text-red-400 disabled:opacity-40"
              aria-label={`Delete scheme: ${s.name}`}
            >
              <Trash2 className="h-3 w-3" aria-hidden="true" />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
