/**
 * @vrooliComponentSource react-component-library:useElementRect
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 5e22d9a4-a909-4a40-8ca2-f3092fa16ffa
 * @vrooliComponentAppliedAt 2026-08-17T08:31:55Z
 * @vrooliComponentSourceSha256 535093099e355ab36dd051aad5a7bf0f3a503bbad454c3fca2bd766c48479e5c
 * @vrooliComponentDriftHash 38ec9df881ab1b662db2076099d36bcb9a7203c39204a7ac92021af89b17fba0
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useEffect, useState } from "react";

export function useElementRect(element: HTMLElement | null) {
  const [rect, setRect] = useState<DOMRectReadOnly | null>(null);
  useEffect(() => {
    if (!element) return;
    const measure = () => setRect(element.getBoundingClientRect());
    measure();
    window.addEventListener("resize", measure);
    return () => window.removeEventListener("resize", measure);
  }, [element]);
  return rect;
}
