import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { fetchDevices, refreshInventory, updateDeviceGroup } from "../api/network";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const panelClass = "rounded-panel border border-app-border bg-app-surface p-4";
const buttonClass = "rounded-control bg-app-primary px-3 py-2 text-sm font-medium text-app-primary-foreground";

export function DevicesPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [findings, setFindings] = useState<string[]>([]);
  const { data = [], isLoading, error } = useQuery({
    queryKey: ["network", "devices"],
    queryFn: () => fetchDevices(""),
  });
  const refresh = useMutation({
    mutationFn: refreshInventory,
    onSuccess: (result) => {
      setFindings(result.findings);
      queryClient.setQueryData(["network", "devices"], result.devices);
    },
  });
  const groupUpdate = useMutation({
    mutationFn: ({ id, group }: { id: string; group: string }) => updateDeviceGroup(id, group),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["network", "devices"] }),
  });

  return (
    <section data-testid={selectors.pages.devices} aria-labelledby="devices-heading" className="flex flex-col gap-4">
      <div>
        <h2 id="devices-heading" className="text-2xl font-semibold">
          {t(strings.pages.devices.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.devices.description)}</p>
      </div>

      <button type="button" className={`${buttonClass} w-fit`} onClick={() => refresh.mutate()}>
        {t(strings.pages.devices.refresh)}
      </button>

      {isLoading && <p data-testid={selectors.network.loading}>{t(strings.network.loading)}</p>}
      {error && <p data-testid={selectors.network.error}>{t(strings.network.error)}</p>}
      {findings.length > 0 && (
        <ul className="list-disc space-y-1 ps-5 text-sm text-app-muted-foreground">
          {findings.map((finding) => <li key={finding}>{finding}</li>)}
        </ul>
      )}

      <section className={panelClass}>
        {data.length === 0 ? (
          <p data-testid={selectors.network.empty} className="text-sm text-app-muted-foreground">
            {t(strings.pages.devices.empty)}
          </p>
        ) : (
          <div data-testid={selectors.network.deviceTable} className="overflow-x-auto">
            <table className="w-full min-w-[48rem] text-left text-sm">
              <thead className="text-app-muted-foreground">
                <tr>
                  <th className="py-2 pe-3 font-medium">{t(strings.pages.devices.hostname)}</th>
                  <th className="py-2 pe-3 font-medium">{t(strings.pages.devices.address)}</th>
                  <th className="py-2 pe-3 font-medium">{t(strings.pages.devices.group)}</th>
                  <th className="py-2 pe-3 font-medium">{t(strings.pages.devices.confidence)}</th>
                  <th className="py-2 font-medium">{t(strings.pages.devices.notes)}</th>
                </tr>
              </thead>
              <tbody>
                {data.map((device) => (
                  <tr key={device.id} className="border-t border-app-border align-top">
                    <td className="py-3 pe-3 font-medium">{device.hostname || t(strings.network.unknown)}</td>
                    <td className="py-3 pe-3">
                      <div>{device.ipAddress || t(strings.network.none)}</div>
                      <div className="text-app-muted-foreground">{device.macAddress || t(strings.network.none)}</div>
                    </td>
                    <td className="py-3 pe-3">
                      <input
                        aria-label={`${t(strings.pages.devices.group)} ${device.hostname || device.id}`}
                        className="w-36 rounded-control border border-app-border bg-app-background px-2 py-1"
                        defaultValue={device.group}
                        onBlur={(event) => {
                          if (event.target.value !== device.group) {
                            groupUpdate.mutate({ id: device.id, group: event.target.value });
                          }
                        }}
                      />
                    </td>
                    <td className="py-3 pe-3">{device.identityConfidence || t(strings.network.unknown)}</td>
                    <td className="py-3">{device.notes.join("; ") || t(strings.network.none)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </section>
  );
}
