import { useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Sparkles, Wand } from "lucide-react";

import { blobUrl } from "../../api/client";
import { LookKind, looksClient, type Look } from "../../api/looks";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

const LOOKS_QUERY_KEY = ["looks"] as const;

/** Result of rendering a Look's preview. */
interface PreviewResult {
  thumbnailRef: string;
  deferredSteps: string[];
}

/**
 * The injected data seam — the network calls LooksView needs. Tests pass a fake
 * so the gallery's render/preview behavior runs without a backend; production
 * uses the generated Connect client.
 */
export interface LooksClient {
  list(): Promise<Look[]>;
  renderPreview(id: string): Promise<PreviewResult>;
}

const defaultLooksClient: LooksClient = {
  list: async () => (await looksClient.listLooks({})).looks,
  renderPreview: async (id) => {
    const r = await looksClient.renderPreview({ lookId: id });
    return { thumbnailRef: r.thumbnailRef, deferredSteps: r.deferredSteps };
  },
};

/**
 * Label key per Look kind. A literal map (not dynamic `strings.looks.kind[…]`)
 * so the unused-key audit can trace each callsite, mirroring aiCatalog's
 * TIER_LABEL.
 */
const KIND_LABEL: Readonly<Record<LookKind, (typeof strings.looks.kind)[keyof typeof strings.looks.kind]>> = {
  [LookKind.STYLE]: strings.looks.kind.style,
  [LookKind.FILM]: strings.looks.kind.film,
  [LookKind.CAMERA]: strings.looks.kind.camera,
  [LookKind.ENHANCE]: strings.looks.kind.enhance,
  [LookKind.CUSTOM]: strings.looks.kind.custom,
  [LookKind.UNSPECIFIED]: strings.looks.kind.unspecified,
};

interface LooksViewProps {
  /** Injected client (tests). Defaults to the live Connect client. */
  client?: LooksClient;
}

/**
 * Looks gallery — browse the Look/Style library (built-in + custom). Each card
 * shows the Look's kind, description, and a preview thumbnail. "Preview" renders
 * the deterministic step chain server-side (film/camera Looks produce an exact
 * preview; style Looks render their deterministic approximation and report the
 * deferred AI steps). Loading / error / empty states are all handled.
 */
export function LooksView({ client = defaultLooksClient }: LooksViewProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  // Rendered previews captured from the mutation result, keyed by Look id. Built-in
  // Looks don't persist their thumbnail server-side, so we hold it locally to show
  // the live preview without mutating the read-only seed entry.
  const [previews, setPreviews] = useState<Readonly<Record<string, PreviewResult>>>({});

  const looksQuery = useQuery({ queryKey: LOOKS_QUERY_KEY, queryFn: () => client.list() });

  const previewMutation = useMutation({
    mutationFn: (id: string) => client.renderPreview(id),
    onSuccess: (result, id) => {
      setPreviews((prev) => ({ ...prev, [id]: result }));
      void queryClient.invalidateQueries({ queryKey: LOOKS_QUERY_KEY });
    },
  });

  let body: ReactNode;
  if (looksQuery.isLoading) {
    body = (
      <p data-testid={selectors.looks.loading} className="text-app-foreground">
        {t(strings.looks.loading)}
      </p>
    );
  } else if (looksQuery.error) {
    body = (
      <p data-testid={selectors.looks.error} className="text-app-danger">
        {errorMessage(looksQuery.error, t)}
      </p>
    );
  } else if ((looksQuery.data ?? []).length === 0) {
    body = <EmptyState testId={selectors.looks.empty} Icon={Sparkles} title={t(strings.looks.empty)} />;
  } else {
    body = (
      <ul
        data-testid={selectors.looks.grid}
        className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4"
      >
        {(looksQuery.data ?? []).map((look, index) => {
          const local = previews[look.id];
          const thumbRef = local?.thumbnailRef || look.thumbnailRef;
          const rendering = previewMutation.isPending && previewMutation.variables === look.id;
          return (
            <li
              key={look.id}
              data-testid={selectors.looks.card({ index: index + 1 })}
              className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-2"
            >
              <div className="relative aspect-square w-full overflow-hidden rounded-control bg-app-muted">
                {thumbRef ? (
                  <img
                    src={blobUrl(thumbRef)}
                    alt={t(strings.looks.thumbnailAlt, { name: look.name })}
                    loading="lazy"
                    className="h-full w-full object-cover animate-develop"
                  />
                ) : (
                  <div className="flex h-full w-full items-center justify-center text-center text-xs text-app-muted-foreground">
                    {t(strings.looks.noPreview)}
                  </div>
                )}
                <span className="absolute left-1.5 top-1.5 rounded-control bg-app-surface/90 px-1.5 py-0.5 text-[10px] font-medium text-app-foreground">
                  {t(KIND_LABEL[look.kind])}
                </span>
                <span className="absolute right-1.5 top-1.5 rounded-control bg-app-surface/90 px-1.5 py-0.5 text-[10px] text-app-muted-foreground">
                  {look.builtin ? t(strings.looks.builtinBadge) : t(strings.looks.customBadge)}
                </span>
              </div>
              <div className="flex flex-col gap-1">
                <span className="truncate text-sm font-medium text-app-foreground">{look.name}</span>
                <p className="line-clamp-2 text-xs text-app-muted-foreground">{look.description}</p>
              </div>
              <Button
                size="sm"
                variant="outline"
                data-testid={selectors.looks.preview({ index: index + 1 })}
                disabled={rendering}
                onClick={() => previewMutation.mutate(look.id)}
              >
                <Wand aria-hidden="true" className="me-2 h-4 w-4" />
                {rendering ? t(strings.looks.rendering) : thumbRef ? t(strings.looks.refresh) : t(strings.looks.preview)}
              </Button>
              {local && local.deferredSteps.length > 0 ? (
                <p className="text-[10px] text-app-muted-foreground">{t(strings.looks.deferred)}</p>
              ) : null}
            </li>
          );
        })}
      </ul>
    );
  }

  return (
    <div data-testid={selectors.looks.root} className="flex flex-col gap-3">
      <p className="text-sm text-app-muted-foreground">{t(strings.looks.description)}</p>
      {body}
    </div>
  );
}
