import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { Button } from "@vrooli/react-component-library/Button/2";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

import { consoleApi, consoleKeys, type Agent } from "../../api/console";
import { AgentMark } from "../../components/console/AgentMark";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

interface StartConversationDialogProps {
  open: boolean;
  onClose: () => void;
}

/**
 * Pick an agent, get a fresh in-app thread. The thread is created through the
 * in-app adapter so the core never learns this surface is special.
 */
export function StartConversationDialog({ open, onClose }: StartConversationDialogProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<string>();
  const roster = useQuery({ queryKey: consoleKeys.agents, queryFn: ({ signal }) => consoleApi.agents(signal), enabled: open });
  const start = useMutation({
    mutationFn: (agentId: string) => consoleApi.startInAppThread(agentId),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: consoleKeys.threads });
      onClose();
      navigate(`/conversations/${created.thread_id}`);
    },
  });
  const agents = (roster.data?.agents ?? []).filter((agent) => !agent.broken);

  return (
    <ResponsiveDialog
      open={open}
      onClose={onClose}
      title={t(strings.console.conversations.startTitle)}
      closeLabel={t(strings.console.common.close)}
      size="sm"
      testId="conversations-start-dialog"
      footer={
        <div className="flex w-full items-center justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose}>
            {t(strings.console.common.cancel)}
          </Button>
          <Button type="button" data-testid="conversations-start-confirm" disabled={!selected} pending={start.isPending} onClick={() => selected && start.mutate(selected)}>
            {t(strings.console.conversations.startConfirm)}
          </Button>
        </div>
      }
    >
      <p className="mb-3 text-sm text-app-muted-foreground">{t(strings.console.conversations.startDescription)}</p>
      {roster.isPending ? <p className="text-sm text-app-muted-foreground">{t(strings.console.region.loading)}</p> : null}
      {roster.isError ? (
        <p role="alert" className="text-sm text-app-danger">
          {roster.error instanceof Error ? roster.error.message : t(strings.errors.unknown)}
        </p>
      ) : null}
      {roster.data && agents.length === 0 ? <p className="text-sm text-app-muted-foreground">{t(strings.console.agents.empty)}</p> : null}
      <div role="radiogroup" aria-label={t(strings.console.agents.title)} className="flex max-h-72 flex-col gap-1 overflow-y-auto">
        {agents.map((agent: Agent) => (
            <button
              key={agent.id}
              type="button"
              role="radio"
              aria-checked={selected === agent.id}
              data-testid="conversations-start-agent"
              onClick={() => setSelected(agent.id)}
              className={[
                "flex w-full items-center gap-3 rounded-panel border px-3 py-2 text-left",
                selected === agent.id ? "border-app-primary bg-app-primary/5" : "border-app-border hover:bg-app-surface-muted",
              ].join(" ")}
            >
              <AgentMark name={agent.display_name} appearance={agent.appearance} size="sm" />
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium text-app-foreground">{agent.display_name}</span>
                {agent.description ? <span className="block truncate text-xs text-app-muted-foreground">{agent.description}</span> : null}
              </span>
            </button>
        ))}
      </div>
      {start.isError ? (
        <p role="alert" className="mt-3 text-sm text-app-danger">
          {start.error instanceof Error ? start.error.message : t(strings.errors.unknown)}
        </p>
      ) : null}
    </ResponsiveDialog>
  );
}
