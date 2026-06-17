import { useState } from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export interface BeforeAfterProps {
  beforeUrl: string;
  afterUrl: string;
}

/**
 * Before/after comparison: the "after" image is clipped to a draggable split
 * controlled by a range slider, revealing the "before" image underneath. The
 * slider is a real <input type="range"> so it is keyboard-accessible and works
 * on touch (mobile). Both images carry alt text via the i18n labels.
 */
export function BeforeAfter({ beforeUrl, afterUrl }: BeforeAfterProps) {
  const { t } = useTranslation();
  const [split, setSplit] = useState(50);

  return (
    <div data-testid={selectors.editor.compare.root} className="flex flex-col gap-2">
      <div className="relative max-h-64 w-full overflow-hidden rounded-lg border border-white/10">
        <img
          data-testid={selectors.editor.compare.before}
          src={beforeUrl}
          alt={t(strings.editor.compare.before)}
          className="block max-h-64 w-full object-contain"
        />
        <img
          data-testid={selectors.editor.compare.after}
          src={afterUrl}
          alt={t(strings.editor.compare.after)}
          className="absolute inset-0 block max-h-64 w-full object-contain"
          style={{ clipPath: `inset(0 ${100 - split}% 0 0)` }}
        />
      </div>
      <label className="flex items-center gap-2 text-xs text-slate-400">
        <span>{t(strings.editor.compare.before)}</span>
        <input
          data-testid={selectors.editor.compare.slider}
          type="range"
          min={0}
          max={100}
          value={split}
          aria-label={t(strings.editor.compare.slider)}
          onChange={(e) => setSplit(Number(e.target.value))}
          className="flex-1"
        />
        <span>{t(strings.editor.compare.after)}</span>
      </label>
    </div>
  );
}
