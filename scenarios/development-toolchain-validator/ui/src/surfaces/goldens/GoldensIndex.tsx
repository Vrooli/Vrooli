import { useState, type FormEvent } from "react";
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Target } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { goldenClient } from "../../api/golden";
import { reportClient } from "../../api/report";
import { errorMessage } from "../../lib/errorMessage";
import { summarizeVerdicts, summaryToVariant } from "../../lib/verdict";
import { ROUTES } from "../../routes.generated";
import { Button } from "../../shared/ui/primitives/Button";
import { Input } from "../../shared/ui/primitives/Input";
import { Card } from "../../shared/ui/primitives/Card";
import { Badge, type BadgeProps } from "../../shared/ui/primitives/Badge";
import {
  Sheet,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "../../shared/ui/primitives/Sheet";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";
import { EmptyState } from "../../shared/ui/composites/EmptyState";
import { LoadingSkeleton } from "../../shared/ui/composites/LoadingSkeleton";

const GOLDENS_QUERY_KEY = ["goldens"] as const;

interface RegisterFormState {
  slug: string;
  template: string;
  version: string;
  path: string;
}

const EMPTY_FORM: RegisterFormState = {
  slug: "",
  template: "",
  version: "",
  path: "",
};

const KIND_TO_VARIANT: Record<string, BadgeProps["variant"]> = {
  pass: "verdict-pass",
  stale: "verdict-stale",
  unexpected: "verdict-unexpected",
  failure: "verdict-failure",
  neutral: "neutral",
};

/**
 * Goldens index surface — first level of the navigation tree.
 *
 * Lists registered goldens, exposes a register-new sheet, and routes to
 * golden detail on row click. Per-row verdict-summary chips fetch from
 * report.getGoldenSummary in parallel via useQueries.
 */
export function GoldensIndex() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [registerOpen, setRegisterOpen] = useState(false);
  const [form, setForm] = useState<RegisterFormState>(EMPTY_FORM);

  const listQuery = useQuery({
    queryKey: GOLDENS_QUERY_KEY,
    queryFn: () => goldenClient.listGoldens({}),
  });

  const registerMutation = useMutation({
    mutationFn: (input: RegisterFormState) => goldenClient.registerGolden(input),
    onSuccess: () => {
      setForm(EMPTY_FORM);
      setRegisterOpen(false);
      void queryClient.invalidateQueries({ queryKey: GOLDENS_QUERY_KEY });
    },
  });

  const handleRegister = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    registerMutation.mutate(form);
  };

  const goldens = listQuery.data?.goldens ?? [];

  const summaryQueries = useQueries({
    queries: goldens.map((g) => ({
      queryKey: ["goldenSummary", g.slug] as const,
      queryFn: () => reportClient.getGoldenSummary({ goldenSlug: g.slug }),
      // Stale summaries are fine; refetch only when the user comes back.
      staleTime: 30_000,
    })),
  });

  return (
    <section
      data-testid={selectors.goldens.card}
      aria-labelledby={selectors.goldens.indexHeading}
      className="flex flex-col gap-4"
    >
      <PanelHeader
        title={<span data-testid={selectors.goldens.indexHeading} id={selectors.goldens.indexHeading}>{t(strings.goldens.title)}</span>}
        description={t(strings.goldens.subtitle)}
        actions={
          <Button
            data-testid={selectors.goldens.registerOpen}
            size="sm"
            onClick={() => setRegisterOpen(true)}
          >
            {t(strings.goldens.registerOpen)}
          </Button>
        }
      />

      {listQuery.isLoading ? (
        <LoadingSkeleton data-testid={selectors.goldens.loading} variant="card" count={3} />
      ) : null}

      {listQuery.error ? (
        <p data-testid={selectors.goldens.error} className="text-sm text-status-failure">
          {errorMessage(listQuery.error, t)}
        </p>
      ) : null}

      {!listQuery.isLoading && goldens.length === 0 && !listQuery.error ? (
        <EmptyState
          testId={selectors.goldens.empty}
          icon={<Target className="h-8 w-8" aria-hidden />}
          title={t(strings.goldens.empty)}
          description={t(strings.goldens.emptyDescription)}
          action={
            <Button size="sm" onClick={() => setRegisterOpen(true)}>
              {t(strings.goldens.registerOpen)}
            </Button>
          }
        />
      ) : null}

      {goldens.length > 0 ? (
        <ul data-testid={selectors.goldens.list} className="flex flex-col gap-2">
          {goldens.map((g, i) => {
            const sQ = summaryQueries[i];
            const summary = sQ?.data?.summary;
            const allTuples = summary
              ? [...summary.skillVerdicts, ...summary.toolVerdicts]
              : [];
            const counts = summarizeVerdicts(allTuples);
            const variantKind = summaryToVariant(counts);
            const chipText = sQ?.isLoading
              ? "…"
              : counts.total === 0
                ? t(strings.goldens.verdictSummaryPending)
                : t(strings.goldens.verdictSummaryCounts, {
                    pass: counts.pass,
                    stale: counts.stale,
                    unexpected: counts.unexpected + counts.failure,
                  });
            return (
              <li key={g.slug}>
                <button
                  type="button"
                  data-testid={selectors.goldens.row}
                  onClick={() => void navigate(ROUTES.goldenDetail(g.slug))}
                  className="w-full text-left"
                >
                  <Card surface="raised" className="transition-colors hover:border-app-accent">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium text-app-foreground">{g.slug}</p>
                        <p className="truncate text-xs text-app-muted-foreground">
                          {t(strings.goldens.rowLabel, {
                            slug: g.slug,
                            template: g.templateId,
                            version: g.templateVersionPinned,
                          })}
                        </p>
                        <p className="mt-1 truncate font-mono text-[11px] text-app-muted-foreground">{g.path}</p>
                      </div>
                      <Badge
                        data-testid={selectors.goldens.rowVerdictSummary}
                        variant={KIND_TO_VARIANT[variantKind] ?? "neutral"}
                      >
                        {chipText}
                      </Badge>
                    </div>
                  </Card>
                </button>
              </li>
            );
          })}
        </ul>
      ) : null}

      <Sheet
        open={registerOpen}
        onOpenChange={setRegisterOpen}
        side="right"
        ariaLabel={t(strings.goldens.registerHeading)}
      >
        <SheetHeader>
          <div>
            <SheetTitle>{t(strings.goldens.registerHeading)}</SheetTitle>
            <SheetDescription>{t(strings.goldens.subtitle)}</SheetDescription>
          </div>
          <Button
            data-testid={selectors.goldens.detailClose}
            size="sm"
            variant="ghost"
            onClick={() => setRegisterOpen(false)}
          >
            {t(strings.goldens.close)}
          </Button>
        </SheetHeader>
        <form
          data-testid={selectors.goldens.registerForm}
          className="mt-4 grid gap-3"
          onSubmit={handleRegister}
        >
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.goldens.slugLabel)}
            <Input
              data-testid={selectors.goldens.registerSlug}
              value={form.slug}
              onChange={(e) => setForm({ ...form, slug: e.target.value })}
              required
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.goldens.templateLabel)}
            <Input
              data-testid={selectors.goldens.registerTemplate}
              value={form.template}
              onChange={(e) => setForm({ ...form, template: e.target.value })}
              required
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.goldens.versionLabel)}
            <Input
              data-testid={selectors.goldens.registerVersion}
              value={form.version}
              onChange={(e) => setForm({ ...form, version: e.target.value })}
              required
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.goldens.pathLabel)}
            <Input
              data-testid={selectors.goldens.registerPath}
              value={form.path}
              onChange={(e) => setForm({ ...form, path: e.target.value })}
              required
            />
          </label>
          <Button
            data-testid={selectors.goldens.registerSubmit}
            type="submit"
            disabled={registerMutation.isPending}
          >
            {registerMutation.isPending
              ? t(strings.goldens.registering)
              : t(strings.goldens.registerSubmit)}
          </Button>
          {registerMutation.error ? (
            <p data-testid={selectors.goldens.registerError} className="text-sm text-status-failure">
              {errorMessage(registerMutation.error, t)}
            </p>
          ) : null}
        </form>
      </Sheet>
    </section>
  );
}
