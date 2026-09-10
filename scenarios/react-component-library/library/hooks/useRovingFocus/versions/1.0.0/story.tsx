import { useRef, useState } from "react";
import { useRovingFocus } from "./useRovingFocus";
export function Default() {
  const first = useRef<HTMLButtonElement>(null);
  const second = useRef<HTMLButtonElement>(null);
  const [active, setActive] = useState(0);
  const onKeyDown = useRovingFocus([first, second], active, setActive);
  return (
    <div data-testid="hooks.use-roving-focus" role="toolbar" tabIndex={0} onKeyDown={onKeyDown}>
      <button ref={first} type="button">
        One
      </button>
      <button ref={second} type="button">
        Two
      </button>
    </div>
  );
}
