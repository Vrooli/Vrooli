/**
 * useDetailUrlSync
 *
 * Bidirectional sync between the detail-selection-store and URL query params.
 *
 * URL params managed:
 *   detail  = "backlog" | "scenario" | "execution" | "initiative"
 *   kind    = backlog kind (only when detail=backlog)
 *   name    = entity name
 *   execId  = execution ID (only when detail=execution)
 *   tab     = active tab within the detail page
 *
 * These live alongside existing graph params (lens, select, focus, returnLens).
 *
 * Sync rules:
 * - On mount: read URL → hydrate store (enables deep-linking)
 * - On store change: mirror to URL (replace, no history push)
 * - On popstate (browser back/forward): read URL → update store
 */

import { useEffect, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import {
  useDetailSelectionStore,
  type DetailSelection,
  type DetailEntityType,
} from "../stores/detail-selection-store";

const DETAIL_ENTITY_TYPES = new Set(["backlog", "scenario", "execution", "initiative", "capture"]);

function isDetailEntityType(value: string | null): value is DetailEntityType {
  return value !== null && DETAIL_ENTITY_TYPES.has(value);
}

/** Read a DetailSelection from URL search params. Returns null if no detail param. */
function readSelectionFromUrl(params: URLSearchParams): DetailSelection | null {
  const detail = params.get("detail");
  if (!isDetailEntityType(detail)) return null;

  const tab = params.get("tab") ?? undefined;

  switch (detail) {
    case "backlog": {
      const kind = params.get("kind");
      const name = params.get("name");
      if (!kind || !name) return null;
      return { entityType: "backlog", kind, name, tab };
    }
    case "scenario": {
      const name = params.get("name");
      if (!name) return null;
      return { entityType: "scenario", name, tab };
    }
    case "execution": {
      const execId = params.get("execId");
      if (!execId) return null;
      return { entityType: "execution", identifier: execId };
    }
    case "capture": {
      const id = params.get("id");
      if (!id) return null;
      return { entityType: "capture", identifier: id };
    }
    case "initiative": {
      const name = params.get("name");
      if (!name) return null;
      return { entityType: "initiative", name, tab };
    }
    default:
      return null;
  }
}

/** Write a DetailSelection to URL search params. Mutates the params in place. */
function writeSelectionToUrl(params: URLSearchParams, selection: DetailSelection | null): void {
  // Always clear detail-related params first.
  params.delete("detail");
  params.delete("kind");
  params.delete("name");
  params.delete("execId");
  params.delete("id");
  params.delete("tab");

  if (!selection) return;

  params.set("detail", selection.entityType);

  switch (selection.entityType) {
    case "backlog":
      if (selection.kind) params.set("kind", selection.kind);
      if (selection.name) params.set("name", selection.name);
      break;
    case "scenario":
    case "initiative":
      if (selection.name) params.set("name", selection.name);
      break;
    case "execution":
      if (selection.identifier) params.set("execId", selection.identifier);
      break;
    case "capture":
      if (selection.identifier) params.set("id", selection.identifier);
      break;
  }

  if (selection.tab) {
    params.set("tab", selection.tab);
  }
}

/** Compare two selections for equality. */
function selectionsEqual(a: DetailSelection | null, b: DetailSelection | null): boolean {
  if (a === b) return true;
  if (!a || !b) return false;
  return (
    a.entityType === b.entityType
    && a.kind === b.kind
    && a.name === b.name
    && a.identifier === b.identifier
    && a.tab === b.tab
  );
}

export function useDetailUrlSync(): void {
  const [searchParams, setSearchParams] = useSearchParams();
  const selection = useDetailSelectionStore((s) => s.selection);
  const hydrate = useDetailSelectionStore((s) => s._hydrate);

  // Track the last selection we wrote to URL to prevent echo loops.
  const lastWritten = useRef<DetailSelection | null>(null);
  // Track whether initial hydration has happened.
  const hydrated = useRef(false);

  // Step 1: On mount, hydrate store from URL (deep-linking).
  useEffect(() => {
    if (hydrated.current) return;
    hydrated.current = true;

    const urlSelection = readSelectionFromUrl(searchParams);
    if (urlSelection) {
      lastWritten.current = urlSelection;
      hydrate(urlSelection);
    }
    // Only run on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Step 2: Mirror store changes to URL.
  useEffect(() => {
    // Skip if this is the same selection we already wrote (prevents echo).
    if (selectionsEqual(selection, lastWritten.current)) return;
    lastWritten.current = selection;

    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        writeSelectionToUrl(next, selection);
        // Clear graph select param when detail is open (detail replaces selection intent).
        if (selection) {
          next.delete("select");
        }
        return next;
      },
      { replace: true },
    );
  }, [selection, setSearchParams]);

  // Step 3: Handle browser back/forward (popstate).
  useEffect(() => {
    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      const urlSelection = readSelectionFromUrl(params);

      if (!selectionsEqual(urlSelection, selection)) {
        lastWritten.current = urlSelection;
        hydrate(urlSelection);
      }
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, [selection, hydrate]);
}
