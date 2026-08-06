/** @vrooliComponentSource hooks.use-focus-visible */
import { useEffect, useState } from "react";

export function useFocusVisible() {
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const onKey = () => setVisible(true);
    const onPointer = () => setVisible(false);
    window.addEventListener("keydown", onKey);
    window.addEventListener("pointerdown", onPointer);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("pointerdown", onPointer);
    };
  }, []);
  return visible;
}
