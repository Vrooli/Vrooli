import { useQuery } from "@tanstack/react-query";
import { ArrowRight } from "lucide-react";
import { Button } from "./components/ui/button";
import { strings } from "./consts/strings";
import { SUPPORTED_LOCALES, getLocaleConfig, setLocale, useTranslation, type Locale } from "./i18n";
import { fetchHealth } from "./lib/api";

export default function App() {
  const { t, i18n } = useTranslation();
  const currentLocale = i18n.language as Locale;

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth
  });

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 flex flex-col items-center justify-center p-6">
      <div className="w-full max-w-xl rounded-2xl border border-white/10 bg-white/5 p-8 shadow-2xl backdrop-blur">
        <div className="flex items-center justify-between gap-4">
          <p className="text-sm uppercase tracking-[0.2em] text-slate-400">{t(strings.app.eyebrow)}</p>
          <div
            role="group"
            aria-label={t(strings.locale.switcherLabel)}
            className="flex items-center gap-1 rounded-full border border-white/10 bg-black/20 p-1 text-xs"
          >
            {SUPPORTED_LOCALES.map((lng) => (
              <button
                key={lng}
                type="button"
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
        <h1 className="mt-3 text-3xl font-semibold">{t(strings.app.title)}</h1>
        <p className="mt-2 text-slate-300">{t(strings.app.description)}</p>

        <div className="mt-6 rounded-xl border border-white/10 bg-black/20 p-4">
          <p className="text-sm font-medium text-slate-400">{t(strings.health.title)}</p>
          {isLoading && <p className="mt-2 text-slate-200">{t(strings.health.loading)}</p>}
          {error && (
            <p className="mt-2 text-red-400">{t(strings.health.error)}</p>
          )}
          {data && (
            <div className="mt-2 text-sm text-slate-200">
              <p>{t(strings.health.statusLabel)} {data.status}</p>
              <p>{t(strings.health.serviceLabel)} {data.service}</p>
              <p>{t(strings.health.timestampLabel)} {new Date(data.timestamp).toLocaleString(currentLocale)}</p>
            </div>
          )}
          <Button className="mt-4" onClick={() => { void refetch(); }}>
            {t(strings.health.refresh)}
            <ArrowRight className="ms-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
