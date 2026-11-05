/**
 * @libraryId react-component-library:useHydrated
 * @displayName useHydrated
 * @description A hydration-awareness primitive that gates browser-only behavior safely, so capability reads never produce a server and client markup mismatch.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useHydrated
 * @vrooliComponentSourceSlot hooks.use-hydrated */
import { useEffect, useState } from "react";

export function useHydrated() {
  const [hydrated, setHydrated] = useState(false);
  useEffect(() => setHydrated(true), []);
  return hydrated;
}
