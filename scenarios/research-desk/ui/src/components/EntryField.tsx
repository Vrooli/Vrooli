import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { forwardRef, useId } from "react";

import { useOverview } from "../api/activity";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { entryLength, entryTooLong } from "../lib/entry";

/**
 * A text field whose words become one entry in an operator file: a running
 * count against the limit the API sent (0 when it could not read it), and, past it, what to do instead (name a file
 * under the research home), before the API has to refuse it.
 */
export const EntryField = forwardRef<HTMLTextAreaElement, { label: string; value: string; limit: number; onChange: (value: string) => void; rows?: number }>(
  function EntryField({ label, value, limit, onChange, rows = 4 }, ref) {
    const { t } = useTranslation();
    const id = useId();
    const countId = useId();
    const home = useOverview().data?.researchHome || t(strings.entry.theHome);
    const used = entryLength(value);
    const over = entryTooLong(value, limit);
    return (
      <div className="flex flex-col gap-space-2xs">
        <label htmlFor={id} className="text-body-sm font-medium">{label}</label>
        <Textarea ref={ref} id={id} rows={rows} value={value} aria-describedby={countId} aria-invalid={over || undefined} onChange={(e) => onChange(e.target.value)} />
        <span id={countId} className={over ? "rd-count rd-count-over" : "rd-count"} aria-live="polite">
          {limit > 0 ? t(strings.entry.count, { used: used.toLocaleString(), limit: limit.toLocaleString() }) : t(strings.entry.countNoLimit, { used: used.toLocaleString() })}
          {over ? <span className="block">{t(strings.entry.tooLong, { home })}</span> : null}
        </span>
      </div>
    );
  },
);
