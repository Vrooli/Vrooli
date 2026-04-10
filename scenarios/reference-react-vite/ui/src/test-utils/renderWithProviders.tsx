// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#ui-test-utilities
// Custom render function that includes application providers
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, RenderOptions } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ReactElement } from 'react';

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  queryClient?: QueryClient;
  initialEntries?: string[];
}

/**
 * Creates a fresh QueryClient configured for testing.
 * Disables retries and garbage collection for deterministic tests.
 */
export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

/**
 * Renders a component with all application providers.
 * Use this instead of render() from @testing-library/react.
 *
 * @param ui - The React element to render
 * @param options - Custom render options including queryClient and initialEntries for routing
 */
export function renderWithProviders(
  ui: ReactElement,
  options: CustomRenderOptions = {}
) {
  const {
    queryClient = createTestQueryClient(),
    initialEntries = ['/'],
    ...renderOptions
  } = options;

  function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <MemoryRouter initialEntries={initialEntries}>
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      </MemoryRouter>
    );
  }

  return {
    queryClient,
    ...render(ui, { wrapper: Wrapper, ...renderOptions }),
  };
}
