import type { Controller, ManualOverride } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { manualOffCoversAll } from "./manualOff";

export function useManualOffDescription(override: ManualOverride | undefined, controllers: Controller[]): string {
  const { t } = useTranslation();
  if (!override?.active) return t(strings.lightDesk.inactive);
  if (manualOffCoversAll(override)) return t(strings.lightDesk.manualOffActive);
  const ids = [...new Set(override.controllerIds.filter((id) => id.trim()))];
  if (!ids.length) return t(strings.lightDesk.manualOffScopeUnknown);
  const scope = ids.map((id) => {
    const controller = controllers.find((item) => item.controllerId === id);
    return controller?.name ? `${controller.name} (${id})` : id;
  }).join(", ");
  return t(strings.lightDesk.manualOffScope, { controllers: scope });
}
