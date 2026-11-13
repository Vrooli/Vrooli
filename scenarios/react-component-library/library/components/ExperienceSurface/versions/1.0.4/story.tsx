import { ExperienceSurface } from "./ExperienceSurface";
import { createElement } from "react";
export function Ready({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(ExperienceSurface, {
    ...args,
    children: createElement("p", {}, "Results are ready"),
  } as never);
}

export function Static({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(ExperienceSurface, {
    ...args,
    children: createElement("p", {}, "Static guidance"),
  } as never);
}

export function Loading({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(ExperienceSurface, {
    ...args,
    children: createElement("p", {}, "Loading results"),
  } as never);
}

export function Empty({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(ExperienceSurface, {
    ...args,
    children: createElement("p", {}, "No results"),
  } as never);
}

export function Partial({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(ExperienceSurface, {
    ...args,
    children: createElement("p", {}, "Some results are unavailable"),
  } as never);
}

export function Error({
  args,
  log,
}: {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
}) {
  void log;
  return createElement(ExperienceSurface, {
    ...args,
    children: createElement("p", {}, "Results could not load"),
  } as never);
}
