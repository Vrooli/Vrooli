import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";

import { cn } from "../lib/utils";
import { chipClass } from "./chipClass";

export interface FilterChip {
  id: string;
  to: string;
  label: string;
  current: boolean;
  testId?: string;
}

const FADE = "pointer-events-none absolute inset-y-0 w-10 md:hidden";

/**
 * One line of filter links that scrolls sideways on a phone and wraps on a wide
 * screen. A fade at whichever edge still has chips beyond it says there is more
 * to scroll to; it disappears once that edge is reached. `testId` names the
 * strip, which also lets a capture see that it is a scroller (its overflow is
 * recorded for declared test ids) and so that the chips off its edge are not
 * page overflow.
 */
export function FilterChips({ label, chips, testId }: { label: string; chips: readonly FilterChip[]; testId?: string }) {
  const strip = useRef<HTMLElement>(null);
  const [more, setMore] = useState({ before: false, after: false });
  const measure = useCallback(() => {
    const element = strip.current;
    if (!element) return;
    // scrollLeft runs negative in a right-to-left page: the distance travelled is its magnitude.
    const travelled = Math.abs(element.scrollLeft);
    const room = element.scrollWidth - element.clientWidth;
    setMore({ before: travelled > 1, after: room - travelled > 1 });
  }, []);
  useEffect(() => {
    measure();
    const element = strip.current;
    if (!element || typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, [measure, chips.length]);

  return (
    <div className="relative -mx-4 md:mx-0">
      <nav
        ref={strip}
        aria-label={label}
        data-testid={testId}
        onScroll={measure}
        className="overflow-x-auto px-4 pb-1 [scrollbar-width:none] md:overflow-visible md:px-0 [&::-webkit-scrollbar]:hidden"
      >
        <ul className="flex w-max gap-2 md:w-auto md:flex-wrap">
          {chips.map((chip) => (
            <li key={chip.id} className="shrink-0">
              <Link
                to={chip.to}
                replace
                data-testid={chip.testId}
                aria-current={chip.current ? "page" : undefined}
                className={chipClass(chip.current)}
              >
                {chip.label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>
      {more.before && <span data-edge="before" aria-hidden="true" className={cn(FADE, "start-0 bg-gradient-to-r from-[var(--color-background)] to-transparent rtl:bg-gradient-to-l")} />}
      {more.after && <span data-edge="after" aria-hidden="true" className={cn(FADE, "end-0 bg-gradient-to-l from-[var(--color-background)] to-transparent rtl:bg-gradient-to-r")} />}
    </div>
  );
}
