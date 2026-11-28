import { render, renderHook, type RenderHookOptions, type RenderHookResult, type RenderOptions, type RenderResult } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";

import { FileViewerProvider, type FileViewerHost } from "../i18n/context.js";

/**
 * Tests render under the `keys` pseudo-locale, where `t(key)` returns the key,
 * so assertions compare against `strings.x.y` instead of translated copy.
 * Pass `host` to render under a real locale or with host preferences.
 */
interface HostOptions { host?: FileViewerHost }

function wrapperFor(host: FileViewerHost | undefined) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <FileViewerProvider locale="keys" {...host}>{children}</FileViewerProvider>;
  };
}

export function renderWithProviders(ui: ReactElement, options: Omit<RenderOptions, "wrapper"> & HostOptions = {}): RenderResult {
  const { host, ...rest } = options;
  return render(ui, { wrapper: wrapperFor(host), ...rest });
}

export function renderHookWithProviders<Result, Props>(
  callback: (props: Props) => Result,
  options: Omit<RenderHookOptions<Props>, "wrapper"> & HostOptions = {},
): RenderHookResult<Result, Props> {
  const { host, ...rest } = options;
  return renderHook(callback, { wrapper: wrapperFor(host), ...rest });
}
