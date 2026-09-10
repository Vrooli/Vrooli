/**
 * @libraryId react-component-library:useScrollLock
 * @displayName useScrollLock
 * @version 2.0.0
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-scroll-lock */
import { useEffect } from "react";

let locks = 0;
let scrollY = 0;
let previous: Partial<CSSStyleDeclaration> = {};

function lockDocument() {
  if (locks++ > 0 || typeof document === "undefined" || typeof window === "undefined") return;
  scrollY = window.scrollY;
  const body = document.body;
  previous = {
    overflow: body.style.overflow,
    position: body.style.position,
    top: body.style.top,
    width: body.style.width,
    paddingRight: body.style.paddingRight,
  };
  const scrollbar = Math.max(0, window.innerWidth - document.documentElement.clientWidth);
  body.style.overflow = "hidden";
  body.style.position = "fixed";
  body.style.top = `-${scrollY}px`;
  body.style.width = "100%";
  if (scrollbar) body.style.paddingRight = `${scrollbar}px`;
}

function unlockDocument() {
  if (locks === 0 || --locks > 0 || typeof document === "undefined" || typeof window === "undefined") return;
  Object.assign(document.body.style, previous);
  window.scrollTo({ top: scrollY, behavior: "instant" });
  previous = {};
}

export function useScrollLock(locked = true) {
  useEffect(() => {
    if (!locked) return;
    lockDocument();
    return unlockDocument;
  }, [locked]);
}

export const scrollLockState = { count: () => locks };
