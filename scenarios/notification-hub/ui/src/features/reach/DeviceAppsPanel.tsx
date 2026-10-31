import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { appDeliveryClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { useTranslation } from "../../i18n";

type Copy = {
  title: string; explanation: string; loading: string; readError: string; retry: string; empty: string;
  chimesail: string; useChimesail: string; removing: string; removed: string; removeError: string; lastUsed: string;
};

const COPY: Record<string, Copy> = {
  en: {
    title: "Apps on your devices",
    explanation: "A scenario's own app can show its notifications on a device instead of Chimesail. Turn that on from the scenario's app, in its notification settings. Each device still gets every notification once.",
    loading: "Loading devices…", readError: "Devices could not be read.", retry: "Try again",
    empty: "No device has notifications turned on yet.",
    chimesail: "Chimesail", useChimesail: "Use Chimesail instead", removing: "Switching…",
    removed: "{app} notifications on {device} now arrive in Chimesail.", removeError: "{app} could not be switched back. Try again.",
    lastUsed: "Last seen {date}",
  },
  ja: {
    title: "デバイスのアプリ",
    explanation: "シナリオ独自のアプリで、Chimesail の代わりにそのデバイスで通知を表示できます。設定はそのアプリの通知設定から行います。各デバイスに届く通知は 1 回だけです。",
    loading: "デバイスを読み込み中…", readError: "デバイスを読み込めませんでした。", retry: "再試行",
    empty: "通知を有効にしたデバイスはまだありません。",
    chimesail: "Chimesail", useChimesail: "Chimesail で受け取る", removing: "切り替え中…",
    removed: "{device} の {app} の通知は Chimesail に届くようになりました。", removeError: "{app} を元に戻せませんでした。再試行してください。",
    lastUsed: "最終確認 {date}",
  },
  ar: {
    title: "التطبيقات على أجهزتك",
    explanation: "يمكن لتطبيق السيناريو نفسه عرض إشعاراته على جهاز بدلًا من Chimesail. فعّل ذلك من إعدادات الإشعارات داخل تطبيق السيناريو. يصل كل إشعار إلى كل جهاز مرة واحدة فقط.",
    loading: "جارٍ تحميل الأجهزة…", readError: "تعذّرت قراءة الأجهزة.", retry: "حاول مرة أخرى",
    empty: "لم يتم تفعيل الإشعارات على أي جهاز بعد.",
    chimesail: "Chimesail", useChimesail: "استخدام Chimesail بدلًا منه", removing: "جارٍ التبديل…",
    removed: "إشعارات {app} على {device} تصل الآن إلى Chimesail.", removeError: "تعذّر إرجاع {app}. حاول مرة أخرى.",
    lastUsed: "آخر ظهور {date}",
  },
};

function fill(template: string, values: Record<string, string>) {
  return template.replace(/\{(\w+)\}/g, (_, key: string) => values[key] ?? "");
}

/**
 * Lists each device that receives notifications and which app shows them
 * there: Chimesail, or a scenario's own app paired from that app's settings.
 * Removing a scenario app hands that device back to Chimesail.
 */
export function DeviceAppsPanel() {
  const { i18n } = useTranslation();
  const c: Copy = COPY[i18n.language.split("-")[0] ?? "en"] ?? (COPY.en as Copy);
  const [message, setMessage] = useState("");
  const devices = useQuery({ queryKey: ["device-apps"], queryFn: () => appDeliveryClient.listDeviceApps({}), refetchOnWindowFocus: true });
  const remove = useMutation({
    mutationFn: async (input: { id: string; app: string; device: string }) => {
      await appDeliveryClient.removeDeviceApp({ subscriptionId: input.id });
      return input;
    },
    onSuccess: (input) => setMessage(fill(c.removed, { app: input.app, device: input.device })),
    onError: (_error, input) => setMessage(fill(c.removeError, { app: input.app })),
    onSettled: () => void devices.refetch(),
  });
  const rows = devices.data?.devices ?? [];
  const date = (iso: string) => {
    const parsed = new Date(iso);
    return Number.isNaN(parsed.getTime()) ? "" : parsed.toLocaleDateString(i18n.language, { month: "short", day: "numeric" });
  };

  return (
    <section data-testid={selectors.deviceApps.root} aria-labelledby="device-apps-title" className="flex flex-col gap-3 rounded-lg border border-app-border p-4">
      <h2 id="device-apps-title" className="text-lg font-semibold">{c.title}</h2>
      <p className="text-sm text-app-muted-foreground">{c.explanation}</p>
      {devices.isPending && <p className="text-sm">{c.loading}</p>}
      {devices.isError && (
        <p className="text-sm">
          {c.readError}{" "}
          <button type="button" className="min-h-11 underline underline-offset-4" onClick={() => void devices.refetch()}>{c.retry}</button>
        </p>
      )}
      {devices.isSuccess && rows.length === 0 && <p className="text-sm">{c.empty}</p>}
      {rows.length > 0 && (
        <ul className="flex flex-col divide-y divide-app-border">
          {rows.map((device) => (
            <li key={device.group} className="flex flex-col gap-2 py-3">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="font-medium">{device.label}</span>
                {date(device.lastUsed) && <span className="text-xs text-app-muted-foreground">{fill(c.lastUsed, { date: date(device.lastUsed) })}</span>}
              </div>
              <ul className="flex flex-col gap-2">
                {device.apps.map((app) => (
                  <li key={app.subscriptionId} className="flex flex-wrap items-center justify-between gap-2 text-sm">
                    <span className="flex min-w-0 flex-col">
                      <span className="font-medium">{app.scenario ? app.displayName : c.chimesail}</span>
                      <span className="text-app-muted-foreground">{app.status}</span>
                    </span>
                    {app.scenario && (
                      <button
                        type="button"
                        data-testid={selectors.deviceApps.removeButton}
                        disabled={remove.isPending}
                        onClick={() => remove.mutate({ id: app.subscriptionId, app: app.displayName, device: device.label })}
                        className="min-h-11 rounded border border-app-border px-3 text-sm font-medium"
                      >
                        {remove.isPending && remove.variables.id === app.subscriptionId ? c.removing : c.useChimesail}
                      </button>
                    )}
                  </li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
      )}
      {message && <p data-testid={selectors.deviceApps.message} role="status" className="text-sm">{message}</p>}
    </section>
  );
}
