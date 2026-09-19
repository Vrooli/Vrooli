import * as React from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";
import { Badge } from "../../components/ui/badge";
import { EmptyState } from "../../components/EmptyState";
import type { FileEntry } from "./lib/graphAdapter";

type KindKey = (typeof strings.explorer.kind)[keyof typeof strings.explorer.kind];

// attributes.kind enum-name → i18n label key. Mirrors the TsNodeKind values
// the ts-morph sidecar emits.
const KIND_LABEL_KEY: Record<string, KindKey> = {
  TS_NODE_KIND_COMPONENT: strings.explorer.kind.component,
  TS_NODE_KIND_HOOK: strings.explorer.kind.hook,
  TS_NODE_KIND_CLASS: strings.explorer.kind.class,
  TS_NODE_KIND_INTERFACE: strings.explorer.kind.interface,
  TS_NODE_KIND_TYPE: strings.explorer.kind.type,
  TS_NODE_KIND_FUNCTION: strings.explorer.kind.function,
  TS_NODE_KIND_VAR: strings.explorer.kind.var,
  TS_NODE_KIND_CONST: strings.explorer.kind.const,
  TS_NODE_KIND_RE_EXPORT: strings.explorer.kind.reExport,
  unknown: strings.explorer.kind.unknown,
};

export interface FileSymbolPanelProps {
  files: readonly FileEntry[];
}

/**
 * File → symbol drill-down. The left column lists files; selecting one reveals
 * its symbols (kind badge + exported flag) in the right column. Self-contained
 * selection state — the rest of the explorer doesn't need to know the choice.
 */
export function FileSymbolPanel({ files }: FileSymbolPanelProps) {
  const { t } = useTranslation();
  const [selectedId, setSelectedId] = React.useState<string | null>(null);

  if (files.length === 0) {
    return (
      <div data-testid={selectors.features.explorer.drilldown.empty}>
        <EmptyState title={t(strings.explorer.files.empty)} />
      </div>
    );
  }

  const selected = files.find((f) => f.id === selectedId) ?? null;

  return (
    <section
      data-testid={selectors.features.explorer.drilldown.root}
      className="grid gap-3 md:grid-cols-2"
    >
      <div className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm">
        <h4 className="text-sm font-semibold">{t(strings.explorer.files.title)}</h4>
        <ul className="flex flex-col gap-1">
          {files.map((file) => {
            const isActive = file.id === selectedId;
            return (
              <li key={file.id}>
                <button
                  type="button"
                  onClick={() => setSelectedId(file.id)}
                  aria-pressed={isActive}
                  className={cn(
                    "flex w-full items-center justify-between gap-2 rounded-control border px-3 py-2 text-left text-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-app-primary",
                    isActive
                      ? "border-app-primary bg-app-surface-muted"
                      : "border-app-border bg-app-surface-muted hover:bg-app-surface",
                  )}
                >
                  <span className="font-mono text-xs text-app-foreground">{file.path}</span>
                  <span className="shrink-0 text-xs text-app-muted-foreground">
                    {t(strings.explorer.files.symbolCount, { count: file.symbols.length })}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      </div>

      <div className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm">
        <h4
          data-testid={selectors.features.explorer.drilldown.title}
          className="text-sm font-semibold"
        >
          {t(strings.explorer.symbols.title)}
        </h4>
        {selected === null ? (
          <p className="text-xs text-app-muted-foreground">{t(strings.explorer.symbols.selectFile)}</p>
        ) : selected.symbols.length === 0 ? (
          <p className="text-xs text-app-muted-foreground">{t(strings.explorer.symbols.empty)}</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {selected.symbols.map((symbol) => (
              <li
                key={symbol.id}
                data-testid={selectors.features.explorer.drilldown.symbol({ id: symbol.id })}
                className="flex flex-col gap-1 rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm"
              >
                <div className="flex items-center gap-2">
                  <Badge variant="outline">
                    {t(KIND_LABEL_KEY[symbol.kind] ?? strings.explorer.kind.unknown)}
                  </Badge>
                  <span className="font-mono text-xs text-app-foreground">{symbol.name}</span>
                  <span className="ms-auto text-xs text-app-muted-foreground">
                    {symbol.exported
                      ? t(strings.explorer.symbols.exported)
                      : t(strings.explorer.symbols.unexported)}
                  </span>
                </div>
                {symbol.leadingComments.length > 0 ? (
                  <div
                    data-testid={selectors.features.explorer.drilldown.symbolComments({
                      id: symbol.id,
                    })}
                  >
                    <p className="sr-only">{t(strings.explorer.symbols.jsdocLabel)}</p>
                    <pre className="overflow-x-auto whitespace-pre-wrap rounded-control border border-app-border bg-app-surface p-2 font-mono text-[11px] leading-snug text-app-muted-foreground">
                      {symbol.leadingComments.join("\n")}
                    </pre>
                  </div>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
