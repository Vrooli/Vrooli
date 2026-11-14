import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

import { LENS, useOverview } from "../api/activity";
import { useNeedsYou } from "../api/operator";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { deliveryOn } from "../lib/needsYou";

/**
 * How the courier is actually doing, in the API's words: its last pass, the
 * last delivery and the last read-back. A courier with every switch off is
 * "off", never a green light.
 */
export function CourierHealth() {
  const { t } = useTranslation();
  const lens = useOverview().data?.lenses.find((l) => l.name === LENS.courier);
  const courier = useNeedsYou().data?.courier;
  if (!lens) return null;
  const off = courier ? !deliveryOn(courier) : false;
  const tone = !lens.available ? "danger" : off ? "neutral" : "success";
  const word = !lens.available ? strings.courier.unhealthy : off ? strings.courier.off : strings.courier.healthy;
  return (
    <div className="rd-row">
      <span className="flex flex-wrap items-center justify-between gap-space-xs">
        <span className="font-semibold">{t(strings.courier.health)}</span>
        <StatusBadge tone={tone}>{t(word)}</StatusBadge>
      </span>
      {lens.reason ? <span className="text-body-sm">{lens.reason}</span> : null}
    </div>
  );
}
