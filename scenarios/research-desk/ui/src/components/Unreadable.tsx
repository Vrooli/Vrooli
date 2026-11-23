import { Alert } from "@vrooli/react-component-library/Alert/1.0.7";
import { List } from "@vrooli/react-component-library/List/1.1.0";

import type { UnparsedHeading } from "@vrooli/proto-types/research-desk/v1/shared/entry_pb";
import type { Evidence } from "@vrooli/proto-types/research-desk/v1/shared/evidence_pb";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { PathText } from "./PathText";

/**
 * Files that exist but cannot be read. What they hold is
 * unknown, so the screen says so instead of letting an empty list read as
 * "nothing there".
 */
export function Unreadable({ evidence }: { evidence?: Evidence[] }) {
  const { t } = useTranslation();
  if (!evidence || evidence.length === 0) return null;
  const title = t(strings.unreadable.title, { count: evidence.length });
  return (
    <Alert
      tone="warning"
      title={title}
      descriptionAs="div"
      description={
        <>
          <span>{t(strings.unreadable.note)}</span>
          <List
            variant="plain"
            label={title}
            items={evidence.map((e, n) => ({
              id: `${e.source}/${n}`,
              title: e.reason || (e.source ? <PathText path={e.source} /> : ""),
              description: e.reason && e.source ? <PathText path={e.source} /> : undefined,
            }))}
          />
        </>
      }
    />
  );
}

/**
 * Headings in a file the desk reads (OPERATOR.md, VERDICTS.md) that do not
 * follow its format: reported with their line, never silently dropped.
 */
export function UnparsedHeadings({ file, headings }: { file: string; headings: UnparsedHeading[] }) {
  const { t } = useTranslation();
  if (headings.length === 0) return null;
  return (
    <Alert
      tone="warning"
      title={t(strings.unreadable.headings, { count: headings.length, file })}
      descriptionAs="div"
      description={
        <>
          <span>{t(strings.unreadable.headingsNote)}</span>
          <ul className="rd-list">
            {headings.map((h) => (
              <li key={h.line}>
                <span className="rd-id">{t(strings.unreadable.line, { line: h.line })}</span> <code>{h.text}</code>
              </li>
            ))}
          </ul>
        </>
      }
    />
  );
}
