import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { AuditTrail } from "@vrooli/react-component-library/AuditTrail/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { strings } from "../../consts/strings";
import { getConfiguration, type Machine, type MachineConfigurationDetail } from "../../api/machines";

/**
 * What has happened to this machine.
 *
 * The audit trail was previously the fourth block down inside the Configure
 * panel, under two collapsed JSON dumps, which is where evidence goes to be
 * ignored. It reads better as its own tab, and separating it also means the
 * Configuration tab is only ever about desired state.
 */
export function MachineActivityTab({ machine }: { machine: Machine }) {
  const { t } = useTranslation();
  const [detail, setDetail] = useState<MachineConfigurationDetail | null>(null);

  useEffect(() => {
    void getConfiguration(machine.target.id)
      .then((result) => setDetail(result.detail))
      .catch(() => setDetail(null));
  }, [machine.target.id]);

  const events = detail?.auditEvents ?? [];

  if (events.length === 0) {
    return (
      <EmptyState
        title={t(strings.machines.activityHeading)}
        description={t(strings.machines.activityEmpty)}
      />
    );
  }

  return (
    <section className="rounded-xl border border-wc-default p-3" data-testid="machine-activity">
      <h3 className="mb-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-wc-text-faint">
        {t(strings.machines.activityHeading)}
      </h3>
      <AuditTrail
        entries={events.map((event) => ({
          actor: event.actor || "system",
          action: [event.action, event.detail].filter(Boolean).join(": "),
        }))}
      />
    </section>
  );
}

export default MachineActivityTab;
