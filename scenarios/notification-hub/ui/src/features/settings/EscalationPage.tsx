import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronDown, ChevronUp, ListOrdered, Plus, X } from "lucide-react";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Button } from "@vrooli/react-component-library/Button/2";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { recipientsClient } from "../../api/notifications";
import { EmptyNotice, QueryError } from "../../components/QueryNotice";
import { ListGroup, ListItem } from "../../components/ListRow";
import { SkeletonRows } from "../../components/Skeleton";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { channelLabel } from "./channelLabel";
import { Notice } from "./Notice";
import { OwnerGate } from "./OwnerGate";
import { devicesQuery, escalationChainQuery, readFresh } from "./ownerQueries";
import { SaveBar } from "./SaveBar";
import { SettingsScreen } from "./SettingsScreen";
import { useSettingsPath } from "./testContext";
import { UnsavedChangesGuard } from "./UnsavedChangesGuard";
import { useSession } from "../session/useSession";

const sameOrder = (a: readonly string[], b: readonly string[]) => a.length === b.length && a.every((channel, index) => channel === b[index]);

const ICON_BUTTON =
  "grid size-11 shrink-0 place-items-center rounded-full text-app-muted-foreground transition-colors duration-quick hover:bg-app-surface-muted hover:text-app-foreground focus-visible:outline focus-visible:outline-2 disabled:opacity-30 disabled:hover:bg-transparent";

export function EscalationPage() {
  const { t } = useTranslation();
  return (
    <SettingsScreen title={t(strings.settings.section.escalation)} description={t(strings.settings.escalation.description)} backToSettings>
      <OwnerGate readOnly={<EscalationContent readOnly />}><EscalationContent /></OwnerGate>
    </SettingsScreen>
  );
}

type Problem = { kind: "mismatch"; actual: string[] } | { kind: "unreadable" };

function EscalationContent({ readOnly = false }: { readOnly?: boolean }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const session = useSession();
  const path = useSettingsPath();
  const chain = useQuery(escalationChainQuery);
  const devices = useQuery({ ...devicesQuery, enabled: session.isSuccess && session.data.signedIn });
  const saved = useMemo(() => chain.data?.steps.map((step) => step.channel) ?? [], [chain.data]);
  // Until the first edit the list simply follows what the Hub holds.
  const [draft, setDraft] = useState<string[] | null>(null);
  const [adding, setAdding] = useState(false);
  const [confirmEmpty, setConfirmEmpty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [problem, setProblem] = useState<Problem | null>(null);
  const [justSaved, setJustSaved] = useState(false);

  const order = draft ?? saved;
  const dirty = draft !== null && !sameOrder(draft, saved);
  const offered = useMemo(() => [...new Set((devices.data?.devices ?? []).flatMap((device) => device.channels))], [devices.data]);
  const addable = offered.filter((channel) => !order.includes(channel));

  const edit = (change: (current: string[]) => string[]) => { if (readOnly) return; setProblem(null); setJustSaved(false); setDraft(change(order)); };
  const move = (index: number, by: -1 | 1) => edit((current) => {
    const next = [...current];
    const moved = next[index];
    const swapped = next[index + by];
    if (moved === undefined || swapped === undefined) return current;
    next[index] = swapped;
    next[index + by] = moved;
    return next;
  });

  const save = async () => {
    if (readOnly) return;
    setConfirmEmpty(false);
    setSaving(true);
    setProblem(null);
    const attempted = order;
    try {
      await recipientsClient.setEscalationChain({ channels: attempted });
    } catch {
      // The read-back below says whether it landed.
    }
    try {
      const latest = await readFresh(queryClient, escalationChainQuery);
      const actual = latest.steps.map((step) => step.channel);
      if (sameOrder(actual, attempted)) { setDraft(null); setJustSaved(true); } else setProblem({ kind: "mismatch", actual });
    } catch {
      setProblem({ kind: "unreadable" });
    }
    setSaving(false);
  };

  if (chain.isPending) return <SkeletonRows rows={2} />;
  if (chain.isError && !chain.data) {
    return <QueryError error={chain.error} title={t(strings.settings.escalation.loadFailed)} detail={t(strings.settings.escalation.loadFailedDetail)} onRetry={() => void chain.refetch()} retrying={chain.isFetching} />;
  }

  return (
    <>
      {readOnly && <Notice>{t(strings.session.readOnlyBrowse)}</Notice>}
      <UnsavedChangesGuard when={dirty} />
      {justSaved && <Notice tone="success" testId={selectors.escalation.message}>{t(strings.settings.escalation.saved)}</Notice>}
      {problem && (
        <Notice tone="danger" title={t(problem.kind === "unreadable" ? strings.settings.escalation.unconfirmedTitle : strings.settings.escalation.notAppliedTitle)} testId={selectors.escalation.problem}>
          {problem.kind === "unreadable"
            ? t(strings.settings.escalation.unconfirmedBody)
            : t(strings.settings.escalation.notAppliedBody, { order: problem.actual.length ? problem.actual.map((channel) => channelLabel(channel, t)).join(t(strings.settings.summary.then)) : t(strings.settings.summary.escalationNone) })}
        </Notice>
      )}
      {order.length === 0 ? (
        <EmptyNotice icon={<ListOrdered size={28} />} title={t(strings.settings.escalation.emptyTitle)} detail={t(strings.settings.escalation.emptyDetail)} />
      ) : (
        <ListGroup aria-label={t(strings.settings.section.escalation)} data-testid={selectors.escalation.list}>
          {order.map((channel, index) => (
            <ListItem key={channel} className="flex items-center gap-3 px-4 py-2" data-testid={selectors.escalation.step({ channel })}>
              <span aria-hidden="true" className="grid size-7 shrink-0 place-items-center rounded-full bg-app-surface-muted text-sm font-semibold">{index + 1}</span>
              <div className="flex min-w-0 flex-1 flex-col py-1">
                <span className="font-semibold [overflow-wrap:anywhere]">{channelLabel(channel, t)}</span>
                {devices.isSuccess && !offered.includes(channel) && <span className="text-sm text-app-warning">{t(strings.settings.escalation.notOffered)}</span>}
              </div>
              {!readOnly && <button type="button" className={ICON_BUTTON} disabled={index === 0 || saving} onClick={() => move(index, -1)} aria-label={t(strings.settings.escalation.moveUp, { channel: channelLabel(channel, t) })} data-testid={selectors.escalation.moveUp({ channel })}>
                <ChevronUp size={20} aria-hidden="true" />
              </button>}
              {!readOnly && <button type="button" className={ICON_BUTTON} disabled={index === order.length - 1 || saving} onClick={() => move(index, 1)} aria-label={t(strings.settings.escalation.moveDown, { channel: channelLabel(channel, t) })} data-testid={selectors.escalation.moveDown({ channel })}>
                <ChevronDown size={20} aria-hidden="true" />
              </button>}
              {!readOnly && <button type="button" className={ICON_BUTTON} disabled={saving} onClick={() => edit((current) => current.filter((item) => item !== channel))} aria-label={t(strings.settings.escalation.remove, { channel: channelLabel(channel, t) })} data-testid={selectors.escalation.remove({ channel })}>
                <X size={20} aria-hidden="true" />
              </button>}
            </ListItem>
          ))}
        </ListGroup>
      )}
      <p className="text-sm text-app-muted-foreground">{t(strings.settings.escalation.timing)}</p>
      {!readOnly && <div>
        <Button className="min-h-11" type="button" variant="outline" disabled={saving} onClick={() => setAdding(true)} data-testid={selectors.escalation.add}>
          <Plus size={18} aria-hidden="true" /> {t(strings.settings.escalation.add)}
        </Button>
      </div>}

      {!readOnly && dirty && (
        <SaveBar>
          <Button className="min-h-11" type="button" variant="primary" disabled={saving} onClick={() => (order.length === 0 ? setConfirmEmpty(true) : void save())} data-testid={selectors.escalation.save}>
            {saving ? t(strings.settings.action.saving) : t(strings.settings.escalation.save)}
          </Button>
          <Button className="min-h-11" type="button" variant="outline" disabled={saving} onClick={() => { setDraft(null); setProblem(null); }} data-testid={selectors.escalation.discard}>
            {t(strings.settings.action.discard)}
          </Button>
        </SaveBar>
      )}

      <ResponsiveDialog open={adding} onClose={() => setAdding(false)} title={t(strings.settings.escalation.addTitle)} closeLabel={t(strings.settings.action.close)} size="sm" contentPadding="comfortable" testId={selectors.escalation.addSheet}>
        {addable.length === 0 ? (
          <div className="flex flex-col gap-3 text-sm">
            <p>{t(strings.settings.escalation.nothingToAdd)}</p>
            <Link to={path("/settings/reach")} className="inline-flex min-h-11 items-center font-medium text-app-primary underline underline-offset-4">{t(strings.settings.escalation.openDevices)}</Link>
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            <p className="text-sm text-app-muted-foreground">{t(strings.settings.escalation.addHint)}</p>
            <ListGroup>
              {addable.map((channel) => (
                <ListItem key={channel}>
                  <button type="button" onClick={() => { edit((current) => [...current, channel]); setAdding(false); }} data-testid={selectors.escalation.choice({ channel })} className="flex min-h-14 w-full items-center justify-between gap-3 px-4 text-start font-medium hover:bg-app-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2">
                    <span className="[overflow-wrap:anywhere]">{channelLabel(channel, t)}</span>
                    <Plus size={18} className="text-app-muted-foreground" aria-hidden="true" />
                  </button>
                </ListItem>
              ))}
            </ListGroup>
          </div>
        )}
      </ResponsiveDialog>

      <AlertDialog
        open={confirmEmpty}
        title={t(strings.settings.escalation.emptyConfirmTitle)}
        description={t(strings.settings.escalation.emptyConfirmBody)}
        destructive
        confirmLabel={t(strings.settings.escalation.emptyConfirm)}
        cancelLabel={t(strings.settings.action.cancel)}
        closeLabel={t(strings.settings.action.cancel)}
        onConfirm={() => void save()}
        onCancel={() => setConfirmEmpty(false)}
      />
    </>
  );
}
