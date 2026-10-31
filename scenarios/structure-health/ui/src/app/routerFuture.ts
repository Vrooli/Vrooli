/**
 * React Router v7 future flags. Opting in early silences the v6 deprecation
 * warnings (which the test setup treats as failures) and keeps routing behavior
 * aligned with v7 before the upgrade lands.
 *
 * Kept in its own leaf module (no other imports) so test providers can pull the
 * flags without dragging the whole route tree / AppShell into every render.
 *
 * Note the v6 split: the data routers (`createBrowserRouter` /
 * `createMemoryRouter`) accept `v7_relativeSplatPath` in their `future` config,
 * while `v7_startTransition` is a `<RouterProvider>` prop. The component
 * `<MemoryRouter>` accepts both via its `future` prop.
 */

/** For the data routers' `future` option. */
export const dataRouterFutureFlags = {
  v7_relativeSplatPath: true,
} as const;

/** For the `<RouterProvider future={...}>` prop. */
export const routerProviderFutureFlags = {
  v7_startTransition: true,
} as const;

/** For the component `<MemoryRouter future={...}>` (accepts both). */
export const routerFutureFlags = {
  v7_startTransition: true,
  v7_relativeSplatPath: true,
} as const;
