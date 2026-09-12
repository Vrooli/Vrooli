import { renderHook, type RenderHookOptions } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

export function renderHookWithProviders<Result, Props>(
  callback: (props: Props) => Result,
  options: RenderHookOptions<Props> = {},
) {
  return renderHook(callback, options);
}

export function createRouterWrapper(initialEntries: string[] = ["/"], initialIndex?: number) {
  return function RouterWrapper({ children }: { children: ReactNode }) {
    return (
      <MemoryRouter
        initialEntries={initialEntries}
        initialIndex={initialIndex}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        {children}
      </MemoryRouter>
    );
  };
}
