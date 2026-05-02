import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight } from "lucide-react";
import { Button } from "./components/ui/button";
import { selectors } from "./consts/selectors";
import { strings } from "./consts/strings";
import { formatDate } from "./i18n/format";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "./i18n";
import { fetchHealth } from "./lib/api";

export default function App() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const [refreshCount, setRefreshCount] = useState(0);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth
  });

  const handleRefresh = () => {
    setRefreshCount((count) => count + 1);
    void refetch();
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 flex flex-col items-center justify-center p-6">
      <div className="w-full max-w-xl rounded-2xl border border-white/10 bg-white/5 p-8 shadow-2xl backdrop-blur">
        <div className="flex items-center justify-between gap-4">
          <p
            data-testid={selectors.app.eyebrow}
            className="text-sm uppercase tracking-[0.2em] text-slate-400"
          >
            {t(strings.app.eyebrow)}
          </p>
          <div
            role="group"
            aria-label={t(strings.locale.switcherLabel)}
            data-testid={selectors.locale.switcher}
            className="flex items-center gap-1 rounded-full border border-white/10 bg-black/20 p-1 text-xs"
          >
            {SUPPORTED_LOCALES.map((lng) => (
              <button
                key={lng}
                type="button"
                data-testid={selectors.locale.toggle({ code: lng })}
                onClick={() => void setLocale(lng)}
                aria-pressed={currentLocale === lng}
                className={
                  currentLocale === lng
                    ? "rounded-full bg-white/15 px-3 py-1 font-medium text-white"
                    : "rounded-full px-3 py-1 text-slate-300 hover:text-white"
                }
              >
                {getLocaleConfig(lng).nativeLabel}
              </button>
            ))}
          </div>
        </div>
        <h1
          data-testid={selectors.app.title}
          className="mt-3 text-3xl font-semibold"
        >
          {t(strings.app.title)}
        </h1>
        <p
          data-testid={selectors.app.description}
          className="mt-2 text-slate-300"
        >
          {t(strings.app.description)}
        </p>

        <div
          data-testid={selectors.health.card}
          className="mt-6 rounded-xl border border-white/10 bg-black/20 p-4"
        >
          <p className="text-sm font-medium text-slate-400">{t(strings.health.title)}</p>
          {isLoading && (
            <p data-testid={selectors.health.loading} className="mt-2 text-slate-200">
              {t(strings.health.loading)}
            </p>
          )}
          {error && (
            <p data-testid={selectors.health.error} className="mt-2 text-red-400">
              {t(strings.health.error)}
            </p>
          )}
          {data && (
            <div className="mt-2 text-sm text-slate-200">
              <p>
                {t(strings.health.statusLabel)}{" "}
                <span data-testid={selectors.health.statusValue}>{data.status}</span>
              </p>
              <p>
                {t(strings.health.serviceLabel)}{" "}
                <span data-testid={selectors.health.serviceValue}>{data.service}</span>
              </p>
              <p>
                {t(strings.health.timestampLabel)}{" "}
                <span data-testid={selectors.health.timestampValue}>
                  {formatDate(new Date(data.timestamp), { dateStyle: "medium", timeStyle: "short" })}
                </span>
              </p>
            </div>
          )}
          <Button
            data-testid={selectors.health.refreshButton}
            className="mt-4"
            onClick={handleRefresh}
          >
            {t(strings.health.refresh)}
            <ArrowRight aria-hidden="true" className="ms-2 h-4 w-4" />
          </Button>
          {refreshCount > 0 && (
            <p
              data-testid={selectors.health.refreshCount}
              className="mt-2 text-xs text-slate-500"
            >
              {t(strings.health.refreshCount, { count: refreshCount })}
            </p>
          )}
          <p
            data-testid={selectors.notifications.summary}
            className="mt-2 text-xs text-slate-500"
          >
            {t(strings.notifications.summary, { count: refreshCount })}
          </p>
        </div>
      </div>
    </div>
  );
}
