/**
 * ToolsContext - Centralized provider for tools state
 *
 * CRITICAL: This context exists to prevent cascading re-renders caused by
 * multiple components calling useTools independently. When each component
 * has its own useTools hook, react-query notifies all subscribers on cache
 * updates, causing exponential re-render cascades that trigger React's
 * "too many re-renders" error (#310).
 *
 * By providing tools data from a single source at the app level, we ensure:
 * 1. Only ONE useTools hook runs (fewer query subscriptions)
 * 2. Derived state is computed once and shared
 * 3. No cascading re-renders between independent useTools instances
 */
import { createContext, useContext, useMemo, ReactNode } from "react";
import { useTools, type UseToolsReturn } from "../hooks/useTools";

// The context type - same as UseToolsReturn but with chatId tracking
interface ToolsContextValue extends UseToolsReturn {
  /** The chatId this context is providing tools for */
  currentChatId: string | undefined;
}

const ToolsContext = createContext<ToolsContextValue | null>(null);

interface ToolsProviderProps {
  /** The chat ID to fetch tools for */
  chatId: string | undefined;
  children: ReactNode;
}

/**
 * Provider component that manages a single useTools instance.
 * Place this at the app level or around the chat view.
 */
export function ToolsProvider({ chatId, children }: ToolsProviderProps) {
  const tools = useTools({ chatId });

  // Memoize the context value to prevent unnecessary re-renders of consumers
  const value = useMemo(
    () => ({
      ...tools,
      currentChatId: chatId,
    }),
    [tools, chatId]
  );

  return (
    <ToolsContext.Provider value={value}>
      {children}
    </ToolsContext.Provider>
  );
}

/**
 * Hook to consume tools data from the context.
 *
 * @param options.chatId - If provided and different from context's chatId,
 *   will fall back to a local useTools call (for modals with different chatId)
 * @returns The tools data from context or local hook
 */
export function useToolsContext(options?: { chatId?: string }): UseToolsReturn {
  const context = useContext(ToolsContext);

  // If context doesn't exist, fall back to direct hook
  // This allows gradual migration and use outside the provider
  const fallbackTools = useTools({
    chatId: options?.chatId,
    // Only enable if we're outside context OR need a different chatId
    enabled: !context || (options?.chatId !== undefined && options.chatId !== context.currentChatId),
  });

  // If no context, use fallback
  if (!context) {
    return fallbackTools;
  }

  // If requesting a different chatId, use fallback
  if (options?.chatId !== undefined && options.chatId !== context.currentChatId) {
    return fallbackTools;
  }

  // Use context value
  return context;
}
