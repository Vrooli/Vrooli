/**
 * @libraryId react-component-library:useFocusVisible
 * @displayName useFocusVisible
 * @description A focus-modality primitive distinguishing keyboard focus from pointer focus so the visible focus treatment appears only when it helps.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useFocusVisible
 * @vrooliComponentSourceSlot hooks.use-focus-visible */
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
