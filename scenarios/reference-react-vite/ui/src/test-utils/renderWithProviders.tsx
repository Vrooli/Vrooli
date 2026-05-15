// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#ui-test-utilities
// Custom render function that includes application providers.
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, RenderOptions } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ReactElement } from 'react';
import { AppContextProvider, type AuthState, type Role } from '../contexts/AppContext';

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  queryClient?: QueryClient;
  initialEntries?: string[];
  auth?: AuthState;
  role?: Role;
  featureBeta?: boolean;
}

export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0, staleTime: 0 },
      mutations: { retry: false },
    },
  });
}

export function renderWithProviders(
  ui: ReactElement,
  options: CustomRenderOptions = {}
) {
  const {
    queryClient = createTestQueryClient(),
    initialEntries = ['/'],
    auth = 'logged_in',
    role = 'viewer',
    featureBeta = false,
    ...renderOptions
  } = options;

  function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <MemoryRouter initialEntries={initialEntries}>
        <QueryClientProvider client={queryClient}>
          <AppContextProvider
            initialAuth={auth}
            initialRole={role}
            initialFeatureBeta={featureBeta}
          >
            {children}
          </AppContextProvider>
        </QueryClientProvider>
      </MemoryRouter>
    );
  }

  return {
    queryClient,
    ...render(ui, { wrapper: Wrapper, ...renderOptions }),
  };
}
