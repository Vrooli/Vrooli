import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { ClipboardCheck, GitCompare, Save } from "lucide-react";
import { useParams } from "react-router-dom";

import {
  applyStudioDraft,
  compareStudioVariants,
  fetchScenarioSpec,
  promoteStudioVariant,
  renderStudioSpec,
  suggestStudioBindings,
  type StudioApplyResult,
} from "../api/experience";
import { PageFrame } from "../components/PageFrame";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Select } from "../components/ui/select";
import { Textarea } from "../components/ui/textarea";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import {
  firstMachineClaim,
  pageDraftFromSpec,
  validationText,
} from "./experiencePageUtils";

export function StudioPage() {
  const { t } = useTranslation();
  const params = useParams();
  const scenario = params.scenario ?? "experience-manager";
  const [selectedPageID, setSelectedPageID] = useState(params.page ?? "");
  const [initializedPageID, setInitializedPageID] = useState("");
  const [title, setTitle] = useState("");
  const [claimStatement, setClaimStatement] = useState("");
  const [result, setResult] = useState<StudioApplyResult>();
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const {
    data: pages,
    isError: specError,
    isLoading: specLoading,
  } = useQuery({
    queryKey: ["experience-studio-spec", scenario],
    queryFn: () => fetchScenarioSpec(scenario),
    staleTime: 60_000,
  });
  const rows = useMemo(() => pages ?? [], [pages]);
  const selectedPage = rows.find((row) => row.document.id === selectedPageID) ?? rows[0];
  const selectedID = selectedPage?.document.id ?? selectedPageID;
  const pageOptions = useMemo(() => {
    if (specLoading) {
      return [{ value: "", label: t(strings.experience.studio.loadingSpec), disabled: true }];
    }
    if (rows.length === 0) {
      return [{ value: "", label: t(strings.experience.studio.emptySpec), disabled: true }];
    }
    return rows.map((row) => ({
      value: row.document.id,
      label: row.document.title || row.spec.page.title,
    }));
  }, [rows, specLoading, t]);

  useEffect(() => {
    if (!selectedPage) {
      return;
    }
    if (initializedPageID === selectedPage.document.id) {
      return;
    }
    setSelectedPageID(selectedPage.document.id);
    setTitle(selectedPage.spec.page.title || selectedPage.document.title);
    setClaimStatement(firstMachineClaim(selectedPage)?.statement ?? t(strings.experience.studio.defaultClaim));
    setResult(undefined);
    setSaveError("");
    setInitializedPageID(selectedPage.document.id);
  }, [initializedPageID, selectedPage, t]);

  const draft = useMemo(
    () => pageDraftFromSpec(selectedPage, title, claimStatement),
    [claimStatement, selectedPage, title],
  );
  const variants = useMemo(
    () => [
      { id: "draft", title: title || selectedPage?.document.title || "draft", page: draft },
      {
        id: "evidence-forward",
        title: t(strings.experience.studio.evidenceForwardVariant),
        page: {
          ...draft,
          title: `${title || selectedPage?.document.title || "draft"} evidence`,
          claims: draft.claims.map((claim) => ({
            ...claim,
            statement: claim.statement || t(strings.experience.studio.emptyClaim),
          })),
        },
      },
    ],
    [draft, selectedPage?.document.title, t, title],
  );
  const renderQuery = useQuery({
    queryKey: ["experience-studio-render", scenario, selectedID],
    queryFn: () => renderStudioSpec(scenario, selectedID),
    enabled: Boolean(selectedID),
    staleTime: 60_000,
  });
  const compareQuery = useQuery({
    queryKey: ["experience-studio-compare", scenario, selectedID, title, claimStatement],
    queryFn: () => compareStudioVariants(scenario, selectedID, variants),
    enabled: Boolean(selectedID && title),
    staleTime: 5_000,
  });
  const suggestionsQuery = useQuery({
    queryKey: ["experience-studio-bindings", scenario, selectedID],
    queryFn: () => suggestStudioBindings(scenario, selectedID),
    enabled: Boolean(selectedID),
    staleTime: 60_000,
  });
  const previewHTML = compareQuery.data?.html || renderQuery.data?.html || "";
  const renderedVariants = compareQuery.data?.variants ?? [];
  const validationCopy = saveError || validationText(result, t(strings.experience.studio.validationCopy));
  const saveDraft = async () => {
    setIsSaving(true);
    setSaveError("");
    try {
      setResult(await applyStudioDraft(scenario, draft));
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : t(strings.errors.unknown));
    } finally {
      setIsSaving(false);
    }
  };
  const promoteDraft = async () => {
    const primaryVariant = variants[0];
    if (!primaryVariant) {
      return;
    }
    setIsSaving(true);
    setSaveError("");
    try {
      setResult(await promoteStudioVariant(scenario, selectedID, primaryVariant));
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : t(strings.errors.unknown));
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <PageFrame
      testId={selectors.pages.studio}
      title={t(strings.experience.studio.title)}
      description={t(strings.experience.studio.description)}
    >
      <div className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
        <form
          data-testid={selectors.experience.studio.specForm}
          aria-label={t(strings.experience.studio.formLabel)}
          className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4"
        >
          <label className="block text-sm font-semibold" htmlFor="studio-page-select">
            {t(strings.experience.common.page)}
          </label>
          <Select
            id="studio-page-select"
            className="mt-2"
            value={selectedID}
            onChange={(event) => {
              setInitializedPageID("");
              setSelectedPageID(event.target.value);
            }}
            disabled={specLoading || rows.length === 0}
            options={pageOptions}
          />
          <label className="mt-4 block text-sm font-semibold" htmlFor="studio-page-title">
            {t(strings.experience.studio.pageTitle)}
          </label>
          <Input
            id="studio-page-title"
            className="mt-2 bg-app-surface text-app-foreground"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            placeholder={t(strings.experience.studio.defaultPage)}
          />
          <label className="mt-4 block text-sm font-semibold" htmlFor="studio-claim">
            {t(strings.experience.common.claims)}
          </label>
          <Textarea
            id="studio-claim"
            className="mt-2 min-h-28 bg-app-surface text-app-foreground"
            value={claimStatement}
            onChange={(event) => setClaimStatement(event.target.value)}
            placeholder={t(strings.experience.studio.emptyClaim)}
          />
          <div
            data-testid={selectors.experience.studio.validationSummary}
            role={saveError ? "alert" : "status"}
            aria-label={t(strings.experience.studio.validationLabel)}
            className="mt-4 whitespace-pre-line rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"
          >
            {specError ? t(strings.experience.studio.loadError) : validationCopy}
          </div>
          <div className="mt-4 rounded-control border border-app-border p-3">
            <p className="text-xs font-semibold uppercase text-app-muted-foreground">
              {t(strings.experience.studio.suggestionsLabel)}
            </p>
            <ul className="mt-2 grid gap-2 text-sm text-app-muted-foreground">
              {(suggestionsQuery.data ?? []).length === 0 ? (
                <li>{suggestionsQuery.isLoading ? t(strings.experience.studio.loadingSuggestions) : t(strings.experience.studio.emptySuggestions)}</li>
              ) : (
                suggestionsQuery.data?.map((suggestion) => (
                  <li key={`${suggestion.elementId}:${suggestion.testid || suggestion.role}`}>
                    <span className="font-medium text-app-foreground">{suggestion.elementId}</span>{" "}
                    {suggestion.testid || suggestion.role || suggestion.accessibleName}
                  </li>
                ))
              )}
            </ul>
          </div>
          <Button
            data-testid={selectors.experience.studio.saveAction}
            type="button"
            className="mt-4"
            onClick={() => void saveDraft()}
            disabled={!selectedID || isSaving}
          >
            <Save className="mr-2 size-4" aria-hidden="true" />
            {isSaving ? t(strings.experience.studio.saving) : t(strings.experience.studio.save)}
          </Button>
        </form>
        <section
          data-testid={selectors.experience.studio.wireframePreview}
          aria-label={t(strings.experience.studio.wireframeLabel)}
          className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.studio.wireframeLabel)}</h3>
          <div className="mt-4 min-h-64 overflow-auto rounded-control border border-dashed border-app-border bg-app-surface-muted p-4">
            {renderQuery.isLoading || compareQuery.isLoading ? (
              <p className="text-sm text-app-muted-foreground">{t(strings.experience.studio.loadingPreview)}</p>
            ) : previewHTML ? (
              <div className="prose prose-sm max-w-full overflow-x-auto text-app-foreground" dangerouslySetInnerHTML={{ __html: previewHTML }} />
            ) : (
              <p className="text-sm text-app-muted-foreground">{t(strings.experience.studio.emptyPreview)}</p>
            )}
          </div>
          <ul
            data-testid={selectors.experience.studio.variantRail}
            aria-label={t(strings.experience.studio.variantsLabel)}
            className="mt-4 grid gap-3 md:grid-cols-2"
          >
            {renderedVariants.length === 0 ? (
              <li className="rounded-control border border-app-border p-3 text-sm text-app-muted-foreground">
                {compareQuery.isError ? t(strings.experience.studio.variantError) : t(strings.experience.studio.emptyVariants)}
              </li>
            ) : (
              renderedVariants.map((variant) => (
              <li key={variant.id} className="rounded-control border border-app-border p-3 text-sm">
                <GitCompare className="mb-2 size-4 text-app-primary" aria-hidden="true" />
                <span className="font-medium">{variant.title}</span>
              </li>
              ))
            )}
          </ul>
          <Button
            data-testid={selectors.experience.studio.promoteAction}
            type="button"
            className="mt-4"
            onClick={() => void promoteDraft()}
            disabled={!selectedID || isSaving}
          >
            <ClipboardCheck className="mr-2 size-4" aria-hidden="true" />
            {t(strings.experience.studio.promote)}
          </Button>
        </section>
      </div>
    </PageFrame>
  );
}
