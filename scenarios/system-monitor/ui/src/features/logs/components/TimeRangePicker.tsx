interface TimeRangePickerProps {
  since: string;
  until: string;
  onChange: (patch: { since?: string; until?: string }) => void;
}

/**
 * Free-text since/until inputs. journalctl accepts both ISO timestamps
 * and human shortcuts (e.g. "10m ago", "yesterday"), so we don't force
 * a date picker — the input passes straight through.
 */
export const TimeRangePicker = ({ since, until, onChange }: TimeRangePickerProps) => (
  <div className="flex-row-center gap-sm">
    <label className="text-xs text-muted" data-sm-style="sm-style-924f08a902">
      Since
      <input
        type="text"
        value={since}
        placeholder="10m ago"
        onChange={(e) => { onChange({ since: e.target.value }); }}
        data-sm-style="sm-style-beca4cd211"
      />
    </label>
    <label className="text-xs text-muted" data-sm-style="sm-style-924f08a902">
      Until
      <input
        type="text"
        value={until}
        placeholder="now"
        onChange={(e) => { onChange({ until: e.target.value }); }}
        data-sm-style="sm-style-beca4cd211"
      />
    </label>
  </div>
);
