/**
 * @libraryId react-component-library:StyleSheet
 * @displayName Style Sheet
 * @description Reusable style sheet foundation implementation for the React component library.
 * @version 1.0.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:StyleSheet */
import { useInsertionEffect } from "react";

type MountedSheet = { css: string; owner: string };

const mountedSheets = new Map<string, MountedSheet>();
const runtimeGlobal: typeof globalThis & { process?: { env?: { NODE_ENV?: string } } } = globalThis;
// eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- Browser globals can omit Node's process/env despite Node typings.
const isDevelopment = runtimeGlobal.process?.env?.NODE_ENV !== "production";

export function normaliseLibraryId(libraryId: string): string {
  return libraryId
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_-]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");
}

export function libraryStyleSheetKey(libraryId: string, version: string): string {
  return `${normaliseLibraryId(libraryId)}-${version.trim()}`;
}

export function useLibraryStyleSheet(first: string, second: string, third?: string): void {
  const key = third === undefined ? first : libraryStyleSheetKey(first, second);
  const css = third === undefined ? second : third;
  const owner = third === undefined ? `legacy:${first}` : `${first}@${second}`;

  useResolvedLibraryStyleSheet(key, css, owner);
}

function useResolvedLibraryStyleSheet(key: string, css: string, owner: string): void {
  useInsertionEffect(() => {
    if (typeof document === "undefined" || !css.trim()) return;
    const escapedKey = key.replace(/[^a-zA-Z0-9_-]/g, "\\$&");
    const selector = `style[data-rcl-sheet="${escapedKey}"]`;
    const existing = document.querySelector<HTMLStyleElement>(selector);
    if (existing) {
      const previous = mountedSheets.get(key);
      const previousCss = previous?.css ?? (existing.textContent || "");
      if (isDevelopment && previousCss !== css) {
        console.error(
          `[react-component-library] stylesheet key collision for ${key}: ${
            previous?.owner ?? "an unknown mounted owner"
          } and ${owner} provide different CSS.`,
        );
      }
      mountedSheets.set(key, { css: previousCss, owner: previous?.owner ?? owner });
      return;
    }
    // A host or test harness may clear the head behind the module-level set.
    // Treat that as a detached sheet and restore it instead of leaving the
    // document without the foundation rules.
    mountedSheets.delete(key);
    const style = document.createElement("style");
    style.dataset.rclSheet = key;
    style.setAttribute("data-rcl-sheet", key);
    style.textContent = css;
    // Library rules must precede consumer styles so an ordinary consumer
    // class wins an equal-specificity cascade tie without render-order tricks.
    document.head.insertBefore(style, document.head.firstChild);
    mountedSheets.set(key, { css, owner });
  }, [key, css]);
}

export interface StyleSheetProps {
  /** @deprecated Pass libraryId and version so the key is collision-proof. */
  name?: string;
  libraryId?: string;
  version?: string;
  css: string;
}

type StyleSheetImplementationProps = Omit<StyleSheetProps, "name"> & { name?: string };

// StyleSheet is the JSX-safe bridge for assets whose style declaration is
// selected by composition. It emits no DOM node; the hook owns one head node
// per key for the whole page.
export function StyleSheet(props: StyleSheetProps): null;
export function StyleSheet({ name, libraryId, version, css }: StyleSheetImplementationProps): null {
  const identity = libraryId?.trim() && version?.trim() ? { libraryId, version } : undefined;
  if (name !== undefined && isDevelopment) {
    console.warn(
      "[react-component-library] StyleSheet.name is deprecated; pass libraryId and version instead.",
    );
  }
  const key = identity ? libraryStyleSheetKey(identity.libraryId, identity.version) : name;
  if (!key && isDevelopment) {
    console.error(
      "[react-component-library] StyleSheet requires libraryId/version or a legacy name.",
    );
  }
  useResolvedLibraryStyleSheet(
    key ?? "",
    css,
    identity ? `${identity.libraryId}@${identity.version}` : `legacy:${key}`,
  );
  return null;
}
