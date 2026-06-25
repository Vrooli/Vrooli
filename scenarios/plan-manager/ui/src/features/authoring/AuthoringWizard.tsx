import { useState } from "react";
import { Link } from "react-router-dom";
import { CheckCircle2, Sparkles, Wand2 } from "lucide-react";

import {
  autofill,
  finalize,
  startSession,
  submitSection,
  validateStructure,
} from "../../api/authoring";
import { SectionPanel } from "../../components/Surfaces";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import type {
  AuthoringSession,
  AutofillResult,
  Section,
  StructureViolation,
} from "@vrooli/proto-types/plan-manager/v1/authoring/authoring_pb";

/** Local wizard state. The session is the source of truth once started. */
interface WizardState {
  session?: AuthoringSession;
  violations: StructureViolation[];
  autofillResults: AutofillResult[];
  finalizedSlug?: string;
  finalizedPlanId?: string;
}

function SectionRow({
  section,
  active,
  onSelect,
}: {
  section: Section;
  active: boolean;
  onSelect: () => void;
}) {
  const { t } = useTranslation();
  return (
    <li data-testid={selectors.authoring.section({ key: section.key })}>
      <button
        type="button"
        onClick={onSelect}
        aria-current={active ? "true" : undefined}
        className={[
          "flex w-full items-center justify-between gap-2 rounded-control border px-3 py-2 text-left text-sm transition-colors",
          active
            ? "border-app-primary bg-app-primary/10 text-app-foreground"
            : "border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
        ].join(" ")}
      >
        <span className="flex items-center gap-2">
          {section.filled ? (
            <CheckCircle2 aria-hidden="true" className="h-4 w-4 text-app-success" />
          ) : (
            <span
              aria-hidden="true"
              className="h-2 w-2 rounded-full bg-app-muted-foreground"
            />
          )}
          {section.label || section.key}
        </span>
        <span className="flex items-center gap-1 text-xs text-app-muted-foreground">
          {section.mandatory ? <span>{t(strings.pages.authoring.mandatory)}</span> : null}
          {section.autofilled ? (
            <span className="rounded-pill bg-app-info/15 px-1.5 text-app-info">
              {t(strings.pages.authoring.autofilled)}
            </span>
          ) : null}
        </span>
      </button>
    </li>
  );
}

/**
 * AuthoringWizard — the guided composer. Start a session, walk + submit
 * sections (surfacing structure violations honestly), run autofill (showing
 * which sources filled and which degraded), and finalize into a structured plan.
 */
export function AuthoringWizard() {
  const { t } = useTranslation();
  const [title, setTitle] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [state, setState] = useState<WizardState>({ violations: [], autofillResults: [] });
  const [activeKey, setActiveKey] = useState("");
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const session = state.session;
  const sections = session?.sections ?? [];
  const activeSection = sections.find((s) => s.key === activeKey);

  const run = async (fn: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await fn();
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };

  const handleStart = (e: React.FormEvent) => {
    e.preventDefault();
    if (title.trim().length === 0) return;
    void run(async () => {
      const s = await startSession(title.trim(), "", templateId.trim());
      if (s) {
        setState({ session: s, violations: [], autofillResults: [] });
        const first = s.currentSectionKey || s.sections[0]?.key || "";
        setActiveKey(first);
        setDraft(s.sections.find((sec) => sec.key === first)?.content ?? "");
      }
    });
  };

  const selectSection = (section: Section) => {
    setActiveKey(section.key);
    setDraft(section.content);
  };

  const handleSubmitSection = () => {
    if (!session || !activeSection) return;
    void run(async () => {
      const res = await submitSection(session.id, activeSection.key, draft);
      if (res.session) {
        setState((prev) => ({ ...prev, session: res.session, violations: res.violations }));
      }
    });
  };

  const handleValidate = () => {
    if (!session) return;
    void run(async () => {
      const res = await validateStructure(session.id);
      setState((prev) => ({ ...prev, violations: res.violations }));
    });
  };

  const handleAutofill = () => {
    if (!session) return;
    void run(async () => {
      const res = await autofill(session.id);
      if (res.session) {
        setState((prev) => ({ ...prev, session: res.session, autofillResults: res.results }));
      }
    });
  };

  const handleFinalize = () => {
    if (!session) return;
    void run(async () => {
      const plan = await finalize(session.id);
      if (plan) {
        setState((prev) => ({
          ...prev,
          finalizedSlug: plan.slug,
          finalizedPlanId: plan.id,
        }));
      }
    });
  };

  if (!session) {
    return (
      <SectionPanel title={t(strings.pages.authoring.startHeading)} headingId="authoring-start-heading">
        <form
          data-testid={selectors.authoring.startForm}
          onSubmit={handleStart}
          className="flex flex-col gap-3"
        >
          <label className="flex flex-col gap-1 text-sm">
            <span className="text-xs font-medium text-app-muted-foreground">
              {t(strings.pages.authoring.startTitleLabel)}
            </span>
            <Input
              data-testid={selectors.authoring.titleInput}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t(strings.pages.authoring.startTitlePlaceholder)}
            />
          </label>
          <label className="flex flex-col gap-1 text-sm">
            <span className="text-xs font-medium text-app-muted-foreground">
              {t(strings.pages.authoring.startTemplateLabel)}
            </span>
            <Input
              data-testid={selectors.authoring.templateInput}
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
            />
          </label>
          {error ? (
            <p role="alert" className="text-sm text-app-danger">
              {errorMessage(error, t)}
            </p>
          ) : null}
          <Button
            type="submit"
            data-testid={selectors.authoring.startButton}
            disabled={busy || title.trim().length === 0}
            className="w-fit"
          >
            {t(strings.pages.authoring.start)}
          </Button>
        </form>
      </SectionPanel>
    );
  }

  if (state.finalizedSlug !== undefined) {
    return (
      <div
        data-testid={selectors.authoring.finalizedBanner}
        role="status"
        className="flex flex-col items-start gap-3 rounded-panel border border-app-success/40 bg-app-success/10 p-6"
      >
        <p className="flex items-center gap-2 text-sm font-medium text-app-success">
          <CheckCircle2 aria-hidden="true" className="h-5 w-5" />
          {t(strings.pages.authoring.finalized, { slug: state.finalizedSlug })}
        </p>
        {state.finalizedPlanId ? (
          <Button asChild variant="outline" size="sm">
            <Link to={`/plans/${state.finalizedPlanId}`}>
              {t(strings.pages.authoring.viewFinalizedPlan)}
            </Link>
          </Button>
        ) : null}
      </div>
    );
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[18rem_1fr]">
      <SectionPanel title={t(strings.pages.authoring.sectionsHeading)} headingId="authoring-sections-heading">
        <ul data-testid={selectors.authoring.sections} className="flex flex-col gap-1.5">
          {sections.map((section) => (
            <SectionRow
              key={section.key}
              section={section}
              active={section.key === activeKey}
              onSelect={() => selectSection(section)}
            />
          ))}
        </ul>
      </SectionPanel>

      <div className="flex flex-col gap-4">
        {error ? (
          <p role="alert" className="rounded-control bg-app-danger/10 px-3 py-2 text-sm text-app-danger">
            {errorMessage(error, t)}
          </p>
        ) : null}

        {activeSection ? (
          <SectionPanel
            title={activeSection.label || activeSection.key}
            headingId="authoring-current-heading"
          >
            <label className="flex flex-col gap-1 text-sm">
              <span className="sr-only">{t(strings.pages.authoring.sectionContentLabel)}</span>
              <Textarea
                data-testid={selectors.authoring.contentInput}
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder={t(strings.pages.authoring.sectionContentPlaceholder)}
                rows={8}
                aria-label={t(strings.pages.authoring.sectionContentLabel)}
              />
            </label>
            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                size="sm"
                data-testid={selectors.authoring.submitButton}
                disabled={busy}
                onClick={handleSubmitSection}
              >
                {t(strings.pages.authoring.submitSection)}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                data-testid={selectors.authoring.validateButton}
                disabled={busy}
                onClick={handleValidate}
              >
                {t(strings.pages.authoring.validateStructure)}
              </Button>
            </div>
          </SectionPanel>
        ) : null}

        <SectionPanel
          title={t(strings.pages.authoring.violationsHeading)}
          headingId="authoring-violations-heading"
        >
          {state.violations.length === 0 ? (
            <p className="flex items-center gap-2 text-sm text-app-success">
              <CheckCircle2 aria-hidden="true" className="h-4 w-4" />
              {t(strings.pages.authoring.noViolations)}
            </p>
          ) : (
            <ul data-testid={selectors.authoring.violations} className="flex flex-col gap-2">
              {state.violations.map((v, i) => (
                <li
                  key={`${v.sectionKey}-${i}`}
                  className="rounded-control border border-app-danger/40 bg-app-danger/10 px-3 py-2 text-sm text-app-danger"
                >
                  <span className="font-mono text-xs">{v.sectionKey}</span>
                  <span className="ms-2">{v.message}</span>
                </li>
              ))}
            </ul>
          )}
        </SectionPanel>

        <SectionPanel
          title={t(strings.pages.authoring.autofillHeading)}
          headingId="authoring-autofill-heading"
          actions={
            <Button
              type="button"
              size="sm"
              variant="outline"
              data-testid={selectors.authoring.autofillButton}
              disabled={busy}
              onClick={handleAutofill}
            >
              <Wand2 aria-hidden="true" className="me-2 h-4 w-4" />
              {t(strings.pages.authoring.autofill)}
            </Button>
          }
        >
          {state.autofillResults.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.authoring.autofillEmpty)}</p>
          ) : (
            <ul data-testid={selectors.authoring.autofillResults} className="flex flex-col gap-1.5">
              {state.autofillResults.map((res, i) => (
                <li
                  key={`${res.source}-${i}`}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm"
                >
                  <span className="font-mono text-xs text-app-foreground">{res.source}</span>
                  <span className="flex items-center gap-2 text-xs">
                    {res.filled ? (
                      <span className="rounded-pill bg-app-success/15 px-2 py-0.5 text-app-success">
                        {t(strings.pages.authoring.filled)}
                      </span>
                    ) : null}
                    {res.degraded ? (
                      <span className="rounded-pill bg-app-warning/15 px-2 py-0.5 text-app-warning">
                        {t(strings.common.degradedBadge)}
                      </span>
                    ) : null}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </SectionPanel>

        <div className="flex justify-end">
          <Button
            type="button"
            data-testid={selectors.authoring.finalizeButton}
            disabled={busy}
            onClick={handleFinalize}
          >
            <Sparkles aria-hidden="true" className="me-2 h-4 w-4" />
            {t(strings.pages.authoring.finalize)}
          </Button>
        </div>
      </div>
    </div>
  );
}
