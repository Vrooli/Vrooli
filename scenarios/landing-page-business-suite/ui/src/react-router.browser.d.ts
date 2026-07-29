import type { NavigateOptions, To } from 'react-router';

interface BrowserNavigateFunction {
  (to: To, options?: NavigateOptions): void;
  (delta: number): void;
}

// LPBS uses BrowserRouter, whose navigate implementation is synchronous. React
// Router intentionally exposes a union to also support data/framework routers;
// narrow it here so type-aware linting reflects the router actually in use.
declare module 'react-router' {
  interface NavigateFunction {
    (to: To, options?: NavigateOptions): void;
    (delta: number): void;
  }
}

// React Router DOM re-exports useNavigate from react-router. Declaring its
// BrowserRouter-specific return type here makes the narrowing visible to
// consumers importing from either package.
declare module 'react-router-dom' {
  export function useNavigate(): BrowserNavigateFunction;
}

export {};
