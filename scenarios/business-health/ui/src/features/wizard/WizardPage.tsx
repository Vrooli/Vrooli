import { useMemo, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { create } from "@bufbuild/protobuf";
import {
  AnswerSchema,
  type Answer,
  type Question,
  type ScaffoldPreview,
  type ScaffoldResult,
  type SessionState,
} from "@vrooli/proto-types/business-health/v1/wizard/wizard_pb";

import { ScenarioPicker } from "../../components/ScenarioPicker";
import { DiffView } from "../../components/DiffView";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { useWizard } from "./useWizard";
import { useRecentScenarios } from "../../hooks/useRecentScenarios";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";

/** Short human rendering of a saved answer for the live section preview. */
const renderAnswer = (answer: Answer): string => {
  if (answer.targets.length > 0) {
    return answer.targets.map((target) => target.title).filter(Boolean).join(", ");
  }
  if (answer.items.length > 0) {
    return answer.items.filter(Boolean).join(", ");
  }
  return answer.text;
};

/**
 * Resumable contract-authoring interview. A scenario picker starts (or resumes)
 * a wizard session; questions are answered one at a time with live validation,
 * a running preview of saved answers, a deterministic scaffold diff, and a
 * gated apply that only unlocks once every required question is answered.
 */
export function WizardPage() {
  const { t } = useTranslation();
  const [scenario, setScenario] = useState("");
  const [reset, setReset] = useState(false);
  const { recents, remember, clear } = useRecentScenarios();
  const { state, preview, result, start, submit, previewScaffold, applyScaffold } = useWizard();

  const choose = (slug: string) => {
    setScenario(slug);
    remember(slug);
    start.mutate({ scenario: slug, reset });
  };

  const currentQuestion = useMemo<Question | null>(() => {
    if (!state || state.complete) return null;
    const nextId = state.remaining[0];
    if (!nextId) return null;
    return state.questions.find((question) => question.id === nextId) ?? null;
  }, [state]);

  const total = state?.questions.length ?? 0;
  const answered = total - (state?.remaining.length ?? 0);

  return (
    <section
      data-testid={selectors.pages.wizard}
      aria-labelledby="wizard-heading"
      className="flex min-h-0 flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="wizard-heading" className="text-2xl font-semibold text-app-foreground">
          {t(strings.wizard.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">{t(strings.wizard.description)}</p>
      </header>

      <div className="flex flex-col gap-2">
        <ScenarioPicker
          onSelect={choose}
          recents={recents}
          onClearRecents={clear}
          initialValue={scenario}
        />
        <label className="flex items-center gap-2 text-xs text-app-muted-foreground">
          <input
            type="checkbox"
            data-testid={selectors.wizard.resetToggle}
            checked={reset}
            onChange={(event) => setReset(event.target.checked)}
            className="h-4 w-4 rounded border-app-border"
          />
          <span>{t(strings.wizard.reset)}</span>
        </label>
      </div>

      {scenario === "" && (
        <p className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground">
          {t(strings.common.chooseScenario)}
        </p>
      )}

      {start.isPending && (
        <p
          data-testid={selectors.wizard.loading}
          className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground"
        >
          {t(strings.wizard.loading)}
        </p>
      )}

      {start.isError && (
        <div
          data-testid={selectors.wizard.error}
          role="alert"
          className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger"
        >
          <p>{t(strings.wizard.error)}</p>
          <p className="mt-1 text-xs opacity-80">{errorMessage(start.error, t)}</p>
        </div>
      )}

      {state && (
        <div className="flex min-h-0 flex-col gap-4 lg:flex-row">
          <div className="flex min-w-0 flex-1 flex-col gap-4">
            <p
              data-testid={selectors.wizard.progress}
              className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground"
            >
              {t(strings.wizard.progress, { answered, total })}
            </p>

            {currentQuestion ? (
              <QuestionForm
                key={currentQuestion.id}
                question={currentQuestion}
                existingAnswer={state.answers[currentQuestion.id]}
                busy={submit.isPending}
                onSubmit={(answer) => submit.mutate([answer])}
              />
            ) : (
              <p className="rounded-panel border border-app-success/40 bg-app-success/10 p-4 text-sm text-app-success">
                {t(strings.wizard.complete)}
              </p>
            )}

            <ScaffoldSection
              busyPreview={previewScaffold.isPending}
              busyApply={applyScaffold.isPending}
              complete={state.complete}
              preview={preview}
              result={result}
              onPreview={() => previewScaffold.mutate()}
              onApply={() => applyScaffold.mutate()}
            />
          </div>

          <div className="flex w-full min-w-0 flex-col gap-4 lg:w-80">
            <SectionPreview state={state} />
            <Hints state={state} />
          </div>
        </div>
      )}
    </section>
  );
}

interface QuestionFormProps {
  readonly question: Question;
  readonly existingAnswer?: Answer;
  readonly busy: boolean;
  readonly onSubmit: (answer: Answer) => void;
}

/**
 * Renders the one active question by kind and builds the typed `Answer` on
 * submit. Local draft state is preserved across a failed submit (invalid
 * answer) because the form only remounts when the question id changes.
 */
function QuestionForm({ question, existingAnswer, busy, onSubmit }: QuestionFormProps) {
  const { t } = useTranslation();
  const [text, setText] = useState(existingAnswer?.text ?? "");
  const [items, setItems] = useState<string[]>(
    existingAnswer?.items.length ? [...existingAnswer.items] : [""],
  );
  const [targets, setTargets] = useState<Array<{ title: string; description: string }>>(
    existingAnswer && existingAnswer.targets.length > 0
      ? existingAnswer.targets.map((target) => ({ title: target.title, description: target.description }))
      : [{ title: "", description: "" }],
  );

  const invalidReason = existingAnswer?.invalidReason ?? "";

  const build = (): Answer => {
    if (question.kind === "list") {
      return create(AnswerSchema, {
        questionId: question.id,
        items: items.map((item) => item.trim()).filter(Boolean),
      });
    }
    if (question.kind === "ot_list") {
      return create(AnswerSchema, {
        questionId: question.id,
        targets: targets
          .map((target) => ({ title: target.title.trim(), description: target.description.trim() }))
          .filter((target) => target.title !== ""),
      });
    }
    return create(AnswerSchema, { questionId: question.id, text: text.trim() });
  };

  return (
    <form
      data-testid={selectors.wizard.question}
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit(build());
      }}
      className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div data-testid={selectors.wizard.step({ id: question.id })} className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="min-w-0 flex-1 text-sm font-medium text-app-foreground">
            {question.prompt}
          </span>
          <span
            className={
              question.required
                ? "rounded-pill bg-app-danger/10 px-2 py-0.5 text-xs font-semibold text-app-danger"
                : "rounded-pill bg-app-surface-muted px-2 py-0.5 text-xs font-semibold text-app-muted-foreground"
            }
          >
            {question.required ? t(strings.wizard.required) : t(strings.wizard.optional)}
          </span>
        </div>
        {question.help && <p className="text-xs text-app-muted-foreground">{question.help}</p>}

        {(question.kind === "text" || question.kind === "multiline") && (
          <Textarea
            data-testid={selectors.wizard.answerText}
            value={text}
            onChange={(event) => setText(event.target.value)}
            rows={question.kind === "multiline" ? 5 : 2}
            className="border-app-border bg-app-surface text-app-foreground placeholder:text-app-muted-foreground focus:ring-app-focus"
          />
        )}

        {question.kind === "list" && (
          <div className="flex flex-col gap-2">
            {items.map((item, index) => (
              <div key={index} className="flex items-center gap-2">
                <Input
                  value={item}
                  onChange={(event) =>
                    setItems((prev) => prev.map((value, i) => (i === index ? event.target.value : value)))
                  }
                  className="border-app-border bg-app-surface text-app-foreground placeholder:text-app-muted-foreground focus:ring-app-focus"
                />
                <button
                  type="button"
                  onClick={() => setItems((prev) => prev.filter((_, i) => i !== index))}
                  aria-label={t(strings.wizard.listRemove)}
                  className="flex h-11 w-11 items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-danger"
                >
                  <Trash2 aria-hidden="true" className="h-4 w-4" />
                </button>
              </div>
            ))}
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setItems((prev) => [...prev, ""])}
              className="self-start"
            >
              <Plus aria-hidden="true" className="me-1 h-4 w-4" />
              {t(strings.wizard.listAdd)}
            </Button>
          </div>
        )}

        {question.kind === "ot_list" && (
          <div data-testid={selectors.wizard.otList} className="flex flex-col gap-3">
            {targets.map((target, index) => (
              <div
                key={index}
                data-testid={selectors.wizard.otEntry({ index })}
                className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-3"
              >
                <Input
                  value={target.title}
                  onChange={(event) =>
                    setTargets((prev) =>
                      prev.map((value, i) => (i === index ? { ...value, title: event.target.value } : value)),
                    )
                  }
                  placeholder={t(strings.wizard.otTitlePlaceholder)}
                  className="border-app-border bg-app-surface text-app-foreground placeholder:text-app-muted-foreground focus:ring-app-focus"
                />
                <Textarea
                  value={target.description}
                  onChange={(event) =>
                    setTargets((prev) =>
                      prev.map((value, i) =>
                        i === index ? { ...value, description: event.target.value } : value,
                      ),
                    )
                  }
                  rows={2}
                  placeholder={t(strings.wizard.otDescPlaceholder)}
                  className="border-app-border bg-app-surface text-app-foreground placeholder:text-app-muted-foreground focus:ring-app-focus"
                />
                <button
                  type="button"
                  onClick={() => setTargets((prev) => prev.filter((_, i) => i !== index))}
                  aria-label={t(strings.wizard.listRemove)}
                  className="flex items-center gap-1 self-start rounded-control px-2 py-1 text-xs text-app-muted-foreground hover:text-app-danger"
                >
                  <Trash2 aria-hidden="true" className="h-4 w-4" />
                  {t(strings.wizard.listRemove)}
                </button>
              </div>
            ))}
            <Button
              type="button"
              variant="outline"
              size="sm"
              data-testid={selectors.wizard.otAdd}
              onClick={() => setTargets((prev) => [...prev, { title: "", description: "" }])}
              className="self-start"
            >
              <Plus aria-hidden="true" className="me-1 h-4 w-4" />
              {t(strings.wizard.listAdd)}
            </Button>
          </div>
        )}

        {invalidReason && (
          <p data-testid={selectors.wizard.invalid} className="text-xs text-app-danger">
            {invalidReason}
          </p>
        )}
      </div>

      <Button data-testid={selectors.wizard.submit} type="submit" disabled={busy} className="self-start">
        {busy ? t(strings.wizard.saving) : t(strings.wizard.submit)}
      </Button>
    </form>
  );
}

interface ScaffoldSectionProps {
  readonly busyPreview: boolean;
  readonly busyApply: boolean;
  readonly complete: boolean;
  readonly preview: ScaffoldPreview | null;
  readonly result: ScaffoldResult | null;
  readonly onPreview: () => void;
  readonly onApply: () => void;
}

/** Preview the deterministic scaffold, then apply it once the answers are complete. */
function ScaffoldSection({
  busyPreview,
  busyApply,
  complete,
  preview,
  result,
  onPreview,
  onApply,
}: ScaffoldSectionProps) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="outline"
          data-testid={selectors.wizard.previewButton}
          disabled={busyPreview}
          onClick={onPreview}
        >
          {busyPreview ? t(strings.wizard.previewLoading) : t(strings.wizard.previewButton)}
        </Button>
        <Button
          type="button"
          data-testid={selectors.wizard.apply}
          disabled={!complete || busyApply}
          onClick={onApply}
        >
          {busyApply ? t(strings.wizard.applying) : t(strings.wizard.apply)}
        </Button>
      </div>

      {preview && (
        <div data-testid={selectors.wizard.preview} className="flex flex-col gap-3">
          <h3 className="text-sm font-semibold text-app-foreground">
            {t(strings.wizard.previewHeading)}
          </h3>
          {preview.files.map((file) => (
            <DiffView key={file.path} before={file.before} after={file.after} path={file.path} />
          ))}
          {preview.blocking.length > 0 && (
            <div className="rounded-control border border-app-warning/40 bg-app-warning/10 p-3 text-xs text-app-warning">
              <p className="font-medium">{t(strings.wizard.previewBlocking)}</p>
              <ul className="mt-1 list-disc ps-5">
                {preview.blocking.map((reason) => (
                  <li key={reason}>{reason}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {result && (
        <div
          data-testid={selectors.wizard.applyResult}
          className="rounded-control border border-app-success/40 bg-app-success/10 p-3 text-xs text-app-success"
        >
          <p className="font-medium">{t(strings.wizard.applied, { count: result.written.length })}</p>
          {result.residualFindings.length > 0 && (
            <>
              <p className="mt-1 font-medium text-app-warning">{t(strings.wizard.residual)}</p>
              <ul className="mt-1 list-disc ps-5 text-app-warning">
                {result.residualFindings.map((finding) => (
                  <li key={finding}>{finding}</li>
                ))}
              </ul>
            </>
          )}
        </div>
      )}
    </div>
  );
}

/** Running list of the answers saved so far (question target + short rendering). */
function SectionPreview({ state }: { readonly state: SessionState }) {
  const { t } = useTranslation();
  const saved = Object.values(state.answers);

  return (
    <div
      data-testid={selectors.wizard.sectionPreview}
      className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-semibold text-app-foreground">
        {t(strings.wizard.sectionPreviewHeading)}
      </h3>
      {saved.length === 0 ? (
        <p className="text-xs text-app-muted-foreground">{t(strings.wizard.sectionPreviewEmpty)}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {saved.map((answer) => {
            const question = state.questions.find((item) => item.id === answer.questionId);
            return (
              <li key={answer.questionId} className="flex flex-col gap-0.5 text-xs">
                <span className="font-mono uppercase tracking-wide text-app-muted-foreground">
                  {question?.target ?? answer.questionId}
                </span>
                <span className="text-app-foreground">{renderAnswer(answer)}</span>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

/** Similar-capability hints surfaced from the search leaf (empty when unavailable). */
function Hints({ state }: { readonly state: SessionState }) {
  const { t } = useTranslation();
  if (state.hints.length === 0) return null;

  return (
    <div
      data-testid={selectors.wizard.hints}
      className="flex flex-col gap-2 rounded-panel border border-app-warning/40 bg-app-warning/10 p-4"
    >
      <h3 className="text-sm font-semibold text-app-warning">{t(strings.wizard.hintsHeading)}</h3>
      <ul className="flex flex-col gap-2">
        {state.hints.map((hint, index) => (
          <li key={`${hint.scenario}-${hint.capability}-${index}`} className="flex flex-col gap-0.5 text-xs">
            <span className="font-medium text-app-foreground">
              {hint.scenario} · {hint.capability}
            </span>
            <span className="font-mono text-app-muted-foreground">{hint.anchor}</span>
            <span className="text-app-muted-foreground">
              {t(strings.wizard.hintScore, { score: hint.score.toFixed(2) })}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
