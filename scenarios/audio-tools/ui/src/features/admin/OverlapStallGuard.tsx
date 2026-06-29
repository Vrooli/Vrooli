import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { getVoiceStreamConfig, updateVoiceStreamConfig } from "../../audio-integration";
import { Panel } from "../../components/ui/panel";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { pushToast } from "../../components/ui/toast";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";

const MIN = 0;
const MAX = 10;

// OverlapStallGuard exposes the overlap_max_stall_rejects admin lever: how
// many consecutive divergence-rejected commit attempts Overlap-Agree
// tolerates before force-committing its longest stable prefix. It rides the
// same StreamConfig update-mask save path as the other overlap levers.
export function OverlapStallGuard() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const config = useQuery({ queryKey: ["stream", "config"], queryFn: getVoiceStreamConfig });

  const [value, setValue] = useState<number | null>(null);

  useEffect(() => {
    if (config.data) setValue(config.data.overlapMaxStallRejects);
  }, [config.data]);

  const save = useMutation({
    mutationFn: (next: number) => updateVoiceStreamConfig({ overlapMaxStallRejects: next }),
    onSuccess: () => {
      pushToast({ title: t(strings.streamConfigAdmin.overlapSaved) });
      void qc.invalidateQueries({ queryKey: ["stream", "config"] });
    },
  });

  return (
    <Panel title={t(strings.streamConfigAdmin.overlapTitle)}>
      <div className="flex flex-col gap-3">
        <p className="text-xs text-app-muted-foreground">{t(strings.streamConfigAdmin.overlapHelp)}</p>

        {config.isPending ? (
          <LoadingRows rows={1} label={t(strings.streamConfigAdmin.overlapTitle)} />
        ) : config.isError ? (
          <ApiErrorState
            error={config.error}
            title={t(strings.streamConfigAdmin.overlapLoadError)}
            onRetry={() => void config.refetch()}
          />
        ) : (
          <>
            <label htmlFor="overlap-stall-rejects" className="flex max-w-xs flex-col gap-1 text-xs">
              {t(strings.streamConfigAdmin.stallRejectsLabel)}
              <Input
                id="overlap-stall-rejects"
                data-testid={selectors.streamConfig.stallRejectsInput}
                type="number"
                min={MIN}
                max={MAX}
                value={value ?? 0}
                onChange={(e) =>
                  setValue(Math.max(MIN, Math.min(MAX, Number(e.currentTarget.value) || 0)))
                }
              />
            </label>
            {value === 0 ? (
              <p className="text-xs text-app-warning">{t(strings.streamConfigAdmin.stallRejectsDisabled)}</p>
            ) : null}
            <div>
              <Button
                type="button"
                data-testid={selectors.streamConfig.saveOverlap}
                disabled={value === null || save.isPending}
                onClick={() => value !== null && save.mutate(value)}
              >
                {t(strings.streamConfigAdmin.saveOverlap)}
              </Button>
            </div>
            {save.isError ? (
              <ApiErrorState
                error={save.error}
                title={t(strings.streamConfigAdmin.overlapError)}
                onRetry={() => value !== null && save.mutate(value)}
              />
            ) : null}
          </>
        )}
      </div>
    </Panel>
  );
}
