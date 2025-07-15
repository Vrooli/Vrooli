import { vi } from "vitest";
import React from "react";

export interface MockRouterConfig {
  pathname?: string;
  search?: string;
  mockNavigation?: boolean;
}

export function createRouterMock(config: MockRouterConfig = {}) {
  const {
    pathname = "/test-path",
    search = "",
    mockNavigation = true,
  } = config;

  const mockSetLocation = vi.fn();
  
  return {
    useLocation: vi.fn(() => [
      { pathname, search },
      mockNavigation ? mockSetLocation : undefined,
    ]),
    useRouter: vi.fn(() => ({
      hook: vi.fn(),
      base: "",
      matcher: vi.fn(() => [false, {}]),
    })),
    useRoute: vi.fn(() => [false, {}]),
    Router: ({ children }: { children: React.ReactNode }) => React.createElement("div", {}, children),
    Route: ({ children }: { children: React.ReactNode }) => React.createElement("div", {}, children),
    Link: ({ children, to, href, onClick, ...props }: { 
      children: React.ReactNode; 
      to?: string; 
      href?: string; 
      onClick?: (e: React.MouseEvent) => void;
      [key: string]: unknown;
    }) => 
      React.createElement("a", { 
        href: to || href || "#", 
        onClick: (e: React.MouseEvent) => {
          e.preventDefault();
          onClick?.(e);
        },
        ...props, 
      }, children),
    Switch: ({ children }: { children: React.ReactNode }) => React.createElement("div", {}, children),
    Redirect: () => null,
    mockSetLocation, // Expose for assertions
  };
}

export const routerMock = createRouterMock();
