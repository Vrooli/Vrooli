import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { ApiErrorState } from "../../../components/composites/ApiErrorState";
import { LoadingRows } from "../../../components/composites/LoadingRows";
import { pushToast } from "../../../components/ui/toast";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { ClipSource, deleteClip, listClips, type ClipMeta } from "../../../services/corpus";

type Translate = (key: string, options?: Record<string, unknown>) => string;

function sourceLabel(t: Translate, source: ClipSource): string {
  switch (source) {
    case ClipSource.FREE_FORM:
      return t(strings.dictationStudio.sourceFreeForm);
    case ClipSource.SCRIPTED:
      return t(strings.dictationStudio.sourceScripted);
    default:
      return t(strings.dictationStudio.sourceUnknown);
  }
}

function formatSeconds(durationMs: number): string {
  return `${(durationMs / 1000).toFixed(1)}s`;
}

// CorpusListView lists stored corpus clips (newest first, from ListClips)
// and supports per-clip deletion.
export function CorpusListView() {
  const { t } = useTranslation();
  const tr = t as unknown as Translate;
  const qc = useQueryClient();
  const clips = useQuery({ queryKey: ["corpus", "clips"], queryFn: () => listClips() });

  const del = useMutation({
    mutationFn: (id: string) => deleteClip(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["corpus", "clips"] });
      pushToast({ title: tr(strings.dictationStudio.clipDeleted) });
    },
  });

  if (clips.isPending) {
    return <LoadingRows rows={3} label={tr(strings.dictationStudio.corpusTitle)} />;
  }
  if (clips.isError) {
    return (
      <ApiErrorState
        error={clips.error}
        title={tr(strings.dictationStudio.corpusError)}
        onRetry={() => void clips.refetch()}
      />
    );
  }

  const list: ClipMeta[] = clips.data;
  if (list.length === 0) {
    return <p className="text-sm text-app-muted-foreground">{tr(strings.dictationStudio.corpusEmpty)}</p>;
  }

  return (
    <ul className="flex flex-col gap-2" data-testid={selectors.dictationStudio.corpusList}>
      {list.map((clip) => (
        <li
          key={clip.id}
          data-testid={selectors.dictationStudio.clipRow({ id: clip.id })}
          className="flex items-start justify-between gap-3 rounded-control border border-app-border px-3 py-2"
        >
          <div className="flex min-w-0 flex-col gap-1">
            <p className="truncate text-sm text-app-foreground">{clip.referenceText}</p>
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="neutral">{sourceLabel(tr, clip.source)}</Badge>
              <span className="text-xs text-app-muted-foreground">{formatSeconds(clip.durationMs)}</span>
              {clip.tags.map((tag) => (
                <Badge key={tag} variant="info">
                  {tag}
                </Badge>
              ))}
            </div>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            data-testid={selectors.dictationStudio.clipDelete({ id: clip.id })}
            disabled={del.isPending}
            onClick={() => del.mutate(clip.id)}
          >
            {t(strings.dictationStudio.deleteClip)}
          </Button>
        </li>
      ))}
    </ul>
  );
}
