/**
 * AppRouter - Root application component using React Router.
 *
 * This is the new simplified entry point for the application.
 * It replaces the 746-line App.tsx with a clean router-based architecture.
 *
 * All routing logic is now handled by:
 * - routes.tsx: Route definitions
 * - views/RootLayout.tsx: Shared layout and providers
 * - views/*View: Individual route components
 *
 * Note: WebSocketProvider, QueryClientProvider, and Toaster are provided
 * by renderApp.tsx to avoid duplicate providers.
 */
import { RouterProvider } from 'react-router-dom';
import { router } from './routes';

/**
 * Root application component.
 *
 * Provides React Router for navigation. Other providers (WebSocket, QueryClient, Toast)
 * are wrapped by renderApp.tsx.
 */
export default function AppRouter() {
  return <RouterProvider router={router} />;
}
