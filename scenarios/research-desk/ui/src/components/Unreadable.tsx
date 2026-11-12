import { Alert } from "@vrooli/react-component-library/Alert/1";

import type { Evidence } from "@vrooli/proto-types/research-desk/v1/shared/evidence_pb";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { PathText } from "./PathText";

/**
 * Files that exist but cannot be read (rulings E1). What they hold is
 * unknown, so the screen says so instead of letting an empty list read as
 * "nothing there".
 */
export function Unreadable({ evidence }: { evidence?: Evidence[] }) {
  const { t } = useTranslation();
  if (!evidence || evidence.length === 0) return null;
  return (
    <Alert
      tone="warning"
      title={t(strings.unreadable.title, { count: evidence.length })}
      description={
        <span className="flex flex-col gap-space-2xs">
          <span>{t(strings.unreadable.note)}</span>
          <ul className="rd-list">
            {evidence.map((e, n) => (
              <li key={`${e.source}/${n}`}>
                {e.reason ? <span className="block">{e.reason}</span> : null}
                {e.source ? <PathText path={e.source} /> : null}
              </li>
            ))}
          </ul>
        </span>
      }
    />
  );
}
