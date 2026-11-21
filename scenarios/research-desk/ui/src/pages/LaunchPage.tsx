import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { Input } from "@vrooli/react-component-library/Input/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { ArrowDown, ArrowUp, Trash2 } from "lucide-react";
import { useEffect, useId, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useCommissions, useTopics, type TopicSummary, type Workstream } from "../api/activity";
import {
  blockersOf,
  launchApi,
  suggestedQuotes,
  useLaunchOptions,
  useLaunchSuggestions,
  usePreflightSpec,
  workstreamWords,
  type GetLaunchOptionsResponse,
  type PacketLimits,
} from "../api/launch";
import { AccessWords } from "../components/AccessWords";
import { CommissionRow } from "../components/CommissionRow";
import { Confirm } from "../components/Confirm";
import { LaunchLink } from "../components/LaunchLink";
import { InlineMarkdown } from "../components/Markdown";
import { PathText } from "../components/PathText";
import { QueryView } from "../components/QueryView";
import { Section } from "../components/Section";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { withRecommendations } from "../lib/choices";
import { launchableAsIs } from "../lib/states";
import { clearSavedForm, emptyForm, LIVE_WORKSTREAM, loadForm, missing, move, saveForm, toSpec, type LaunchForm } from "../lib/spec";
import { matching, sortByResearchWrite } from "../lib/topics";
import { useDebounced } from "../lib/useDebounced";
import { ChoiceSummary, MessageText, PreflightResult } from "./launch/Preflight";

/**
 * Launch: choose topics and their cards, give your words, allowance
 * and models, and see the packet's gates, the verdict on earlier commissions,
 * and the two messages before one Launch. When automatic launch is not
 * available the same screen ends in a manual start instead.
 */
export function LaunchPage() {
  const { t } = useTranslation();
  const options = useLaunchOptions();
  return (
    <div data-testid={selectors.pages.launch} className="rd-page">
      <PageHeader headingId="launch-heading" title={t(strings.launch.title)} description={t(strings.launch.description)} />
      <PreparedPackets />
      <QueryView query={options}>
        {(o) => (o.packetLimits ? <Builder options={o} limits={o.packetLimits} /> : <Alert tone="danger" title={t(strings.launch.limitsUnknown)} />)}
      </QueryView>
    </div>
  );
}

function PreparedPackets() {
  const { t } = useTranslation();
  const prepared = useCommissions().data?.commissions.filter(launchableAsIs) ?? [];
  if (prepared.length === 0) return null;
  return (
    <Section id="prepared" title={t(strings.launch.prepared, { count: prepared.length })} note={t(strings.launch.preparedNote)}>
      <ul className="rd-rows">
        {prepared.map((c) => (
          <CommissionRow key={c.id} commission={c}>
            <LaunchLink id={c.id} />
          </CommissionRow>
        ))}
      </ul>
    </Section>
  );
}

function Builder({ options, limits }: { options: GetLaunchOptionsResponse; limits: PacketLimits }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const topics = useTopics();
  const defaults = { supervisorModel: options.defaultSupervisorModel, supervisorEffort: options.defaultSupervisorEffort, ownerModelTier: options.defaultOwnerModelTier, allowanceHours: limits.allowanceHoursDefault };
  const [form, setForm] = useState<LaunchForm>(() => loadForm(defaults, options));
  useEffect(() => saveForm(form), [form]);
  const [confirm, setConfirm] = useState<"auto" | "manual" | null>(null);
  const update = (patch: Partial<LaunchForm>) => setForm((prev) => ({ ...prev, ...patch }));

  // The desk's suggested quotes fill the list until the operator edits one.
  const suggestions = useLaunchSuggestions(form.topics);
  const suggested = suggestedQuotes(suggestions.data);
  const first = suggested.quotes[0]?.source ?? "";
  const texts = suggested.quotes.map((q) => q.text).join("\n");
  useEffect(() => {
    if (!texts) return;
    setForm((prev) => {
      const typed = prev.quotes.some((q) => q.trim()) && !prev.quotesFrom;
      if (typed || (prev.quotesFrom === first && prev.quotes.join("\n") === texts)) return prev;
      return { ...prev, quotes: texts.split("\n"), quotesFrom: first };
    });
  }, [texts, first]);
  const titled = new Map((suggestions.data?.topicWorkstreams ?? []).map((x) => [x.topic, x.workstreams]));

  const gaps = missing(form, limits);
  const specJson = useMemo(() => (gaps.length === 0 ? JSON.stringify(toSpec(form)) : null), [form, gaps.length]);
  const checkedSpec = useDebounced(specJson, 600);
  const preflight = usePreflightSpec(checkedSpec, form.choices);
  const result = specJson ? preflight.data : undefined;
  const packet = result?.packet;
  const byName = new Map((topics.data?.topics ?? []).map((x) => [x.topic, x]));
  const blockers = result ? blockersOf(result) : [];

  // The desk's recommendations arrive selected; the operator's own choice is never overwritten.
  useEffect(() => {
    if (!result) return;
    setForm((prev) => {
      const choices = withRecommendations(blockersOf(result), prev.choices);
      const readFirst = { ...prev.readFirst };
      let changed = false;
      for (const v of result.topics) {
        if (v.readFirst && !(v.topic in readFirst) && !prev.readFirstDeclined.includes(v.topic) && prev.topics.includes(v.topic)) {
          readFirst[v.topic] = v.readFirst;
          changed = true;
        }
      }
      return choices || changed ? { ...prev, choices: choices ?? prev.choices, readFirst } : prev;
    });
  }, [result]);

  // Launch only what the preview shows: not while a newer edit awaits its check.
  const ready = Boolean(result?.launchable && packet?.ok) && !preflight.isFetching && checkedSpec === specJson;

  const build = async () => {
    const res = await launchApi.buildFromSpec(JSON.stringify(toSpec(form)), form.choices);
    if (!res.packet?.ok || !res.packet.commissionId) {
      throw new Error(res.packet?.errors.map((e) => e.message).join("; ") || t(strings.launch.buildFailed));
    }
    clearSavedForm();
    return res.packet.commissionId;
  };

  return (
    <>
      {!options.automaticAvailable ? (
        <Alert tone="info" title={t(strings.launch.manualModeTitle)} description={t(strings.launch.manualMode, { reason: options.automaticReason || t(strings.common.unknown) })} />
      ) : null}
      <Section id="step-topics" title={t(strings.launch.stepTopics)} note={t(strings.launch.stepTopicsNote, { max: limits.topicsMax })}>
        <QueryView query={topics}>
          {({ topics: all }) => <TopicPicker all={all} max={limits.topicsMax} chosen={form.topics} onChange={(next) => update({ topics: next })} />}
        </QueryView>
      </Section>
      <Section id="step-cards" title={t(strings.launch.stepCards)} note={t(strings.launch.stepCardsNote, { max: limits.orderedWorkstreamsMax })}>
        {form.topics.length === 0 ? (
          <p className="text-body-sm text-app-muted-foreground">{t(strings.launch.cardsAfterTopics)}</p>
        ) : (
          <ol className="rd-rows">
            {form.topics.map((name) => (
              <CardEditor
                key={name}
                topic={byName.get(name)}
                titled={titled.get(name)}
                max={limits.orderedWorkstreamsMax}
                name={name}
                order={form.order[name] ?? []}
                note={form.notes[name] ?? ""}
                missionOrder={packet?.cards.find((c) => c.topic === name)?.order ?? []}
                onOrder={(ids) => update({ order: { ...form.order, [name]: ids } })}
                onNote={(text) => update({ notes: { ...form.notes, [name]: text } })}
              />
            ))}
          </ol>
        )}
      </Section>
      <Section id="step-words" title={t(strings.launch.stepWords)} note={t(strings.launch.stepWordsNote)}>
        <Words form={form} update={update} reason={suggested.reason} />
      </Section>
      <Section id="step-run" title={t(strings.launch.stepRun)}>
        <RunSettings form={form} update={update} options={options} limits={limits} />
      </Section>
      <Section id="step-check" title={t(strings.launch.stepCheck)} note={t(strings.launch.stepCheckNote)}>
        {gaps.length > 0 ? (
          <ul className="rd-list text-body-sm">
            {gaps.map((gap) => (
              <li key={gap}>{t(GAP[gap], { min: limits.allowanceHoursMin, max: limits.allowanceHoursMax })}</li>
            ))}
          </ul>
        ) : preflight.isError ? (
          <Alert tone="danger" title={t(strings.launch.checkFailed)} description={preflight.error.message} />
        ) : !result ? (
          <p className="text-body-sm">{t(strings.launch.checking)}</p>
        ) : (
          <>
            {preflight.isFetching ? <p className="text-caption text-app-muted-foreground">{t(strings.launch.checking)}</p> : null}
            <PreflightResult
              result={result}
              decide={{
                blockers,
                choices: form.choices,
                onChoose: (id, choice) => update({ choices: { ...form.choices, [id]: choice } }),
                readFirst: form.readFirst,
                onReadFirst: (topic, path) => {
                  const rest = Object.fromEntries(Object.entries(form.readFirst).filter(([key]) => key !== topic));
                  update(path ? { readFirst: { ...rest, [topic]: path } } : { readFirst: rest, readFirstDeclined: [...form.readFirstDeclined, topic] });
                },
              }}
            />
            {packet?.texts ? (
              <details className="rd-fold">
                <summary>{t(strings.launch.messages)}</summary>
                <p className="text-body-sm text-app-muted-foreground">{t(strings.launch.messagesNote)}</p>
                <MessageText label={t(strings.launch.stance)} text={packet.texts.stance} />
                <MessageText label={t(strings.launch.kernel)} text={packet.texts.kernel} hint={t(strings.launch.kernelHint)} />
                <details className="rd-fold">
                  <summary>{t(strings.launch.charter, { kb: Math.round(new TextEncoder().encode(packet.texts.charter).length / 100) / 10 })}</summary>
                  <pre className="rd-exact">{packet.texts.charter}</pre>
                </details>
              </details>
            ) : null}
          </>
        )}
        <div>
          <Button variant="ghost" onClick={() => setForm(emptyForm(defaults))}>{t(strings.launch.startOver)}</Button>
        </div>
      </Section>
      <div className={form.topics.length > 0 ? "rd-launch-bar rd-launch-bar-sticky" : "rd-launch-bar"}>
        {options.automaticAvailable ? (
          <>
            <Button variant="primary" disabled={!ready} onClick={() => setConfirm("auto")} data-testid={selectors.launch.launchButton}>
              {t(strings.launch.launch)}
            </Button>
            <Button variant="ghost" disabled={!ready} onClick={() => setConfirm("manual")}>{t(strings.launch.buildManual)}</Button>
          </>
        ) : (
          <Button variant="primary" disabled={!ready} onClick={() => setConfirm("manual")}>{t(strings.launch.buildManual)}</Button>
        )}
      </div>
      <Confirm
        open={confirm === "auto"}
        title={t(strings.launch.confirmTitle)}
        description={t(strings.launch.confirm, { id: packet?.commissionId ?? "", model: form.supervisorModel, effort: form.supervisorEffort })}
        confirmLabel={t(strings.launch.launch)}
        onClose={() => setConfirm(null)}
        onConfirm={async () => {
          const id = await build();
          let launchError = "";
          try {
            await launchApi.launch(id);
          } catch (err) {
            launchError = err instanceof Error ? err.message : String(err);
          }
          navigate(`/launch/${encodeURIComponent(id)}`, { state: { launchError } });
        }}
      >
        <ol className="rd-list text-body-sm">
          <li>{t(strings.launch.confirmStep1, { id: packet?.commissionId ?? "" })}</li>
          <li>{t(strings.launch.confirmStep2)}</li>
          <li>{t(strings.launch.confirmStep3)}</li>
          <li>{t(strings.launch.confirmStep4)}</li>
        </ol>
        <ChoiceSummary choices={form.choices} />
        <p className="text-body-sm">
          <span className="font-semibold">{t(strings.launch.access)}</span> <AccessWords kind={options.webConsoleAccessKind} said={options.webConsoleAccess} />
        </p>
      </Confirm>
      <Confirm
        open={confirm === "manual"}
        title={t(strings.launch.confirmManualTitle)}
        description={t(strings.launch.confirmManual, { id: packet?.commissionId ?? "" })}
        confirmLabel={t(strings.launch.buildManual)}
        onClose={() => setConfirm(null)}
        onConfirm={async () => {
          const id = await build();
          navigate(`/launch/${encodeURIComponent(id)}?manual=1`);
        }}
      >
        <ChoiceSummary choices={form.choices} />
      </Confirm>
    </>
  );
}

const GAP = {
  topics: strings.launch.needTopics,
  quotes: strings.launch.needQuotes,
  allowance: strings.launch.needAllowance,
} as const;

/** How many topics show before "Show all": the most recently written first. */
const PICK_SHOWN = 8;

function TopicPicker({ all, max, chosen, onChange }: { all: TopicSummary[]; max: number; chosen: string[]; onChange: (next: string[]) => void }) {
  const { t } = useTranslation();
  const [filter, setFilter] = useState("");
  const [all_, setAll] = useState(false);
  const sorted = sortByResearchWrite(matching(all, filter));
  // Chosen topics always show; the rest are cut at PICK_SHOWN unless filtering or expanded.
  const shown = filter || all_ ? sorted : sorted.filter((x, n) => n < PICK_SHOWN || chosen.includes(x.topic));
  const hidden = sorted.length - shown.length;
  return (
    <div className="flex flex-col gap-space-sm">
      <Input type="search" value={filter} onChange={(e) => setFilter(e.target.value)} aria-label={t(strings.launch.filterTopics)} placeholder={t(strings.launch.filterTopics)} />
      <ul className="rd-pick">
        {shown.map((x) => {
          const on = chosen.includes(x.topic);
          return (
            <li key={x.topic}>
              <Checkbox
                label={x.title?.value || x.topic}
                description={[x.topic, x.missionCheck?.mode, windowOpen(x) ? t(strings.launch.topicRunning) : ""].filter(Boolean).join(" · ")}
                checked={on}
                disabled={!on && chosen.length >= max}
                onCheckedChange={(value) => onChange(value ? [...chosen, x.topic] : chosen.filter((c) => c !== x.topic))}
              />
            </li>
          );
        })}
      </ul>
      {hidden > 0 ? (
        <div>
          <Button variant="secondary" onClick={() => setAll(true)}>{t(strings.launch.showAllTopics, { count: sorted.length })}</Button>
        </div>
      ) : null}
    </div>
  );
}

function CardEditor(props: {
  name: string;
  topic: TopicSummary | undefined;
  /** The workstreams with their titles, from the launch options; the topic list's own until they arrive. */
  titled: Workstream[] | undefined;
  /** How many workstreams an order may hold. */
  max: number;
  order: string[];
  note: string;
  missionOrder: string[];
  onOrder: (ids: string[]) => void;
  onNote: (text: string) => void;
}) {
  const { t } = useTranslation();
  const noteId = useId();
  const live = (props.titled ?? props.topic?.missionCheck?.workstreams ?? []).filter((w) => LIVE_WORKSTREAM.has(w.status));
  const unpicked = live.filter((w) => !props.order.includes(w.id));
  const all = props.titled ?? props.topic?.missionCheck?.workstreams ?? [];
  const titleOf = (id: string) => {
    const w = all.find((x) => x.id === id);
    return w ? workstreamWords(w).title : "";
  };
  return (
    <li className="rd-row">
      <span className="font-semibold">{props.topic?.title?.value || props.name}</span>
      <span className="text-body-sm text-app-muted-foreground">
        {props.order.length === 0
          ? t(strings.launch.missionOrder, { order: props.missionOrder.join(", ") || t(strings.common.unknown) })
          : t(strings.launch.yourOrder)}
      </span>
      {props.order.length > 0 ? (
        <ol className="rd-order">
          {props.order.map((id, n) => (
            <li key={id}>
              <span className="rd-num">{n + 1}</span>
              <span className="min-w-0 flex-1">
                <span className="rd-clamp font-medium">
                  <InlineMarkdown text={titleOf(id) || id} />
                </span>
                <span className="rd-id block text-caption text-app-muted-foreground">{id}</span>
              </span>
              <IconButton aria-label={t(strings.launch.moveUp, { id })} disabled={n === 0} onClick={() => props.onOrder(move(props.order, n, -1))}>
                <ArrowUp aria-hidden="true" />
              </IconButton>
              <IconButton aria-label={t(strings.launch.moveDown, { id })} disabled={n === props.order.length - 1} onClick={() => props.onOrder(move(props.order, n, 1))}>
                <ArrowDown aria-hidden="true" />
              </IconButton>
              <IconButton aria-label={t(strings.launch.removeWorkstream, { id })} onClick={() => props.onOrder(props.order.filter((x) => x !== id))}>
                <Trash2 aria-hidden="true" />
              </IconButton>
            </li>
          ))}
        </ol>
      ) : null}
      {unpicked.length > 0 && props.order.length < props.max ? (
        <details className="rd-fold">
          <summary>{t(strings.launch.addWorkstream)}</summary>
          <ul className="rd-rows" aria-label={t(strings.launch.addWorkstream)}>
            {unpicked.map((w) => {
              const words = workstreamWords(w);
              return (
                <li key={w.id}>
                  <button type="button" className="rd-pick-row" onClick={() => props.onOrder([...props.order, w.id])} aria-label={t(strings.launch.addId, { id: w.id, status: w.status })}>
                    <span className="rd-clamp font-semibold">
                      <InlineMarkdown text={words.title || w.id} />
                    </span>
                    <span className="text-caption text-app-muted-foreground">
                      <span className="rd-id">{w.id}</span> · {w.status}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        </details>
      ) : live.length === 0 ? (
        <span className="text-body-sm">{t(strings.launch.noLive)}</span>
      ) : null}
      <label htmlFor={noteId} className="text-body-sm font-medium">{t(strings.launch.cardNote)}</label>
      <Textarea id={noteId} rows={2} value={props.note} onChange={(e) => props.onNote(e.target.value)} />
    </li>
  );
}

function Words({ form, update, reason }: { form: LaunchForm; update: (patch: Partial<LaunchForm>) => void; reason: string }) {
  const { t } = useTranslation();
  const emphasisId = useId();
  // Editing a suggested quote makes the list the operator's own: it is no longer replaced.
  const setQuotes = (quotes: string[]) => update({ quotes, quotesFrom: "" });
  return (
    <div className="flex flex-col gap-space-sm">
      <label htmlFor={emphasisId} className="text-body-sm font-medium">{t(strings.launch.emphasis)}</label>
      <Textarea id={emphasisId} rows={3} value={form.emphasis} onChange={(e) => update({ emphasis: e.target.value })} />
      <span className="text-body-sm font-medium">{t(strings.launch.quotes)}</span>
      {form.quotesFrom ? (
        <span className="text-caption text-app-muted-foreground">
          {t(strings.launch.quotesFrom)} <PathText path={form.quotesFrom} />
        </span>
      ) : !form.quotes.some((q) => q.trim()) && reason ? (
        <span className="text-caption text-app-muted-foreground">{reason}</span>
      ) : null}
      {form.quotes.map((quote, n) => (
        <div key={n} className="flex items-start gap-space-xs">
          <Textarea
            rows={2}
            aria-label={t(strings.launch.quoteN, { n: n + 1 })}
            value={quote}
            onChange={(e) => setQuotes(form.quotes.map((q, i) => (i === n ? e.target.value : q)))}
            className="min-w-0 flex-1"
          />
          {form.quotes.length > 1 ? (
            <IconButton aria-label={t(strings.launch.removeQuote, { n: n + 1 })} onClick={() => setQuotes(form.quotes.filter((_, i) => i !== n))}>
              <Trash2 aria-hidden="true" />
            </IconButton>
          ) : null}
        </div>
      ))}
      <div>
        <Button variant="secondary" onClick={() => setQuotes([...form.quotes, ""])}>{t(strings.launch.addQuote)}</Button>
      </div>
    </div>
  );
}

function RunSettings({ form, update, options, limits }: { form: LaunchForm; update: (patch: Partial<LaunchForm>) => void; options: GetLaunchOptionsResponse; limits: PacketLimits }) {
  const { t } = useTranslation();
  const hoursId = useId();
  const slugId = useId();
  const costlier = (list: string[], chosen: string, base: string) => list.indexOf(chosen) > list.indexOf(base);
  const ownerBase = options.defaultOwnerModelTier;
  const warnings = [
    costlier(options.supervisorModels, form.supervisorModel, options.defaultSupervisorModel) && t(strings.launch.costModel, { model: form.supervisorModel, base: options.defaultSupervisorModel }),
    costlier(options.supervisorEfforts, form.supervisorEffort, options.defaultSupervisorEffort) && t(strings.launch.costEffort, { effort: form.supervisorEffort, base: options.defaultSupervisorEffort }),
    form.ownerModelTier !== ownerBase && costlier(options.supervisorModels, form.ownerModelTier, options.defaultSupervisorModel) && t(strings.launch.costOwner, { model: form.ownerModelTier }),
  ].filter(Boolean) as string[];
  return (
    <div className="rd-form-grid">
      <label htmlFor={hoursId} className="flex flex-col gap-space-2xs text-body-sm">
        <span className="font-medium">{t(strings.launch.allowance)}</span>
        <Input id={hoursId} type="number" inputMode="numeric" min={limits.allowanceHoursMin} max={limits.allowanceHoursMax} value={String(form.allowanceHours)} onChange={(e) => update({ allowanceHours: Number(e.target.value) })} />
        <span className="text-caption text-app-muted-foreground">{t(strings.launch.allowanceHint, { min: limits.allowanceHoursMin, max: limits.allowanceHoursMax })}</span>
      </label>
      <label className="flex flex-col gap-space-2xs text-body-sm">
        <span className="font-medium">{t(strings.launch.supervisorModel)}</span>
        <Select
          value={form.supervisorModel}
          onValueChange={(value) => update({ supervisorModel: value })}
          options={options.supervisorModels.map((m) => ({ value: m, label: m === options.defaultSupervisorModel ? t(strings.launch.defaultOption, { name: m }) : m }))}
        />
      </label>
      <label className="flex flex-col gap-space-2xs text-body-sm">
        <span className="font-medium">{t(strings.launch.supervisorEffort)}</span>
        <Select
          value={form.supervisorEffort}
          onValueChange={(value) => update({ supervisorEffort: value })}
          options={options.supervisorEfforts.map((m) => ({ value: m, label: m === options.defaultSupervisorEffort ? t(strings.launch.defaultOption, { name: m }) : m }))}
        />
      </label>
      <label className="flex flex-col gap-space-2xs text-body-sm">
        <span className="font-medium">{t(strings.launch.ownerTier)}</span>
        <Select
          value={form.ownerModelTier}
          onValueChange={(value) => update({ ownerModelTier: value })}
          options={options.ownerModelTiers.map((m) => ({ value: m, label: m === "inherit" ? t(strings.launch.inherit) : m === ownerBase ? t(strings.launch.defaultOption, { name: m }) : m }))}
        />
      </label>
      <label htmlFor={slugId} className="flex flex-col gap-space-2xs text-body-sm">
        <span className="font-medium">{t(strings.launch.slug)}</span>
        <Input id={slugId} value={form.slug} onChange={(e) => update({ slug: e.target.value.toLowerCase() })} placeholder={t(strings.launch.slugPlaceholder)} autoCapitalize="none" spellCheck={false} />
        <span className="text-caption text-app-muted-foreground">{t(strings.launch.slugHint)}</span>
      </label>
      {warnings.map((w) => (
        <Alert key={w} tone="warning" title={t(strings.launch.costTitle)} description={w} />
      ))}
    </div>
  );
}

/** A supervisor window is open on this topic (live, paused or limited; never summed elsewhere). */
function windowOpen(topic: TopicSummary): boolean {
  return topic.liveCommissionIds.length + topic.pausedCommissionIds.length + topic.limitedCommissionIds.length > 0;
}
