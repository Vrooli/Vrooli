import { Button } from "@vrooli/react-component-library/Button/2.2.0";
import { Select } from "@vrooli/react-component-library/Select/1.1.1";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { useQuery } from "@tanstack/react-query";
import { archiveGoal, configuredBookId, fetchBooks, fetchGoals, reparentGoal } from "../api/ledger";
import { formatCurrency } from "../i18n/format";
import { useMutation, useQueryClient } from "@tanstack/react-query";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
const THEME_LABEL_KEYS = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
};

/**
 * Settings page. Surfaces the locale and theme selectors as a real page (in
 * addition to the compact controls in the top bar). Add scenario-specific
 * preferences here as they're needed.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const queryClient = useQueryClient();
  const books = useQuery({ queryKey: ["settings-books"], queryFn: fetchBooks, retry: false });
  const bookId = configuredBookId() || books.data?.books[0]?.id || "";
  const goals = useQuery({ queryKey: ["settings-goals", bookId], queryFn: () => fetchGoals(bookId), retry: false, enabled: Boolean(bookId) });
  const goalRows = goals.data?.goals ?? [];
  const archiveGoalMutation = useMutation({
    mutationFn: (goalId: string) => archiveGoal(goalId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["settings-goals", bookId] }),
  });
  const reparentGoalMutation = useMutation({
    mutationFn: (input: { goalId: string; targetBookId: string }) => reparentGoal(input.goalId, input.targetBookId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["settings-goals", bookId] }),
  });

  return (
    <ExperienceSurface
      surfaceId="settings"
      state="ready"
      data-testid={selectors.pages.settings}
      aria-labelledby="settings-heading"
      className="flex flex-col gap-6"
    >
      <h2 id="settings-heading" className="text-2xl font-semibold">
        {t(strings.pages.settings.title)}
      </h2>

      <section className="rounded-md border p-4" aria-labelledby="settings-goals-heading">
        <h3 id="settings-goals-heading" className="font-semibold">{t(strings.pages.settings.goalsHeading)}</h3>
        <ul data-testid="settings-goal-list" aria-label={t(strings.pages.settings.goalsHeading)} className="mt-2 text-sm text-app-muted-foreground">
          {goalRows.length === 0 ? <li>{t(strings.pages.settings.goalsEmpty)}</li> : goalRows.map((verdict) => {
            const goal = verdict.goal;
            const targetBook = books.data?.books.find((book) => book.id !== bookId);
            if (!goal) return null;
            return <li key={goal.id} className="flex flex-wrap items-center gap-2">
              <span>{t(strings.pages.settings.goalSummary, { name: goal.name, threshold: formatCurrency(Number(goal.thresholdMinor) / 100, books.data?.books.find((book) => book.id === bookId)?.currency || "USD"), comparator: goal.comparator, periods: goal.sustainPeriods })}</span>
              <Button type="button" size="sm" variant="secondary" data-testid="settings-goal-archive" disabled={archiveGoalMutation.isPending} onClick={() => archiveGoalMutation.mutate(goal.id)}>{t(strings.pages.settings.archiveGoal)}</Button>
              {targetBook && <Button type="button" size="sm" variant="secondary" data-testid="settings-goal-reparent" disabled={reparentGoalMutation.isPending} onClick={() => reparentGoalMutation.mutate({ goalId: goal.id, targetBookId: targetBook.id })}>{t(strings.pages.settings.moveGoal)}</Button>}
            </li>;
          })}
        </ul>
        <div className="mt-3 grid gap-2 sm:grid-cols-3">
          <span data-testid="settings-goal-threshold" role="group" aria-label={t(strings.pages.settings.goalThreshold)} className="rounded border p-2 text-sm">{goalRows.length ? t(strings.pages.settings.goalThreshold) : t(strings.pages.settings.goalsEmpty)}</span>
          <span data-testid="settings-goal-sustain-window" role="group" aria-label={t(strings.pages.settings.goalSustainWindow)} className="rounded border p-2 text-sm">{goalRows.length ? t(strings.pages.settings.goalSustainWindow) : t(strings.pages.settings.goalsEmpty)}</span>
          <span data-testid="settings-goal-buffer" role="group" aria-label={t(strings.pages.settings.goalBuffer)} className="rounded border p-2 text-sm">{goalRows.length ? t(strings.pages.settings.goalBuffer) : t(strings.pages.settings.goalsEmpty)}</span>
        </div>
        <p data-testid="settings-goals-empty-guidance" role="note" className="mt-2 text-sm text-app-muted-foreground">
          {goalRows.length ? t(strings.pages.settings.goalDescription) : t(strings.pages.settings.goalsEmpty)}
        </p>
      </section>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.themeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.theme.switcherLabel)} data-testid="settings-theme" className="flex flex-wrap gap-2">
          {THEME_CHOICES.map((c) => (
            <Button
              key={c}
              type="button"
              variant={choice === c ? "primary" : "secondary"}
              size="sm"
              role="radio"
              aria-checked={choice === c}
              onClick={() => setTheme(c)}
              data-testid={selectors.settingsPage.themeOption({ choice: c })}
            >
              {t(THEME_LABEL_KEYS[c])}
            </Button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.localeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} data-testid="settings-locale" className="flex flex-wrap gap-2">
          {SUPPORTED_LOCALES.map((lng) => (
            <Button
              key={lng}
              type="button"
              variant={currentLocale === lng ? "primary" : "secondary"}
              size="sm"
              role="radio"
              aria-checked={currentLocale === lng}
              onClick={() => void setLocale(lng)}
              data-testid={selectors.settingsPage.localeOption({ code: lng })}
            >
              {getLocaleConfig(lng).nativeLabel}
            </Button>
          ))}
        </div>
      </div>

      <div className="grid gap-2 sm:grid-cols-2">
        <label className="flex flex-col gap-1 rounded border p-2 text-sm">
          <span>{t(strings.pages.settings.defaultBook)}</span>
          <Select
            data-testid="settings-default-book"
            aria-label={t(strings.pages.settings.defaultBook)}
            defaultValue="none"
            className="min-h-11 rounded border bg-background px-2 py-1"
            options={[{ value: "none", label: t(strings.pages.settings.defaultBook) }]}
          />
        </label>
        <label className="flex flex-col gap-1 rounded border p-2 text-sm">
          <span>{t(strings.pages.settings.currencyDisplay)}</span>
          <Select
            data-testid="settings-currency-display"
            aria-label={t(strings.pages.settings.currencyDisplay)}
            defaultValue="separate"
            className="min-h-11 rounded border bg-background px-2 py-1"
            options={[{ value: "separate", label: t(strings.pages.settings.currencyDisplay) }]}
          />
        </label>
      </div>
    </ExperienceSurface>
  );
}
