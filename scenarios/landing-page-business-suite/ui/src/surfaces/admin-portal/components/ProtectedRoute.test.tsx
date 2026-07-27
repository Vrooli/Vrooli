import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderWithProviders as render } from "../../../test-utils/renderWithProviders";
import { screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { ProtectedRoute } from './ProtectedRoute';
import * as useAdminAuthModule from '../../../app/providers/useAdminAuth';

const renderWithAuth = (isAuthenticated: boolean) => {
  // Mock the useAdminAuth hook to return our desired state
  vi.spyOn(useAdminAuthModule, 'useAdminAuth').mockReturnValue({
    isAuthenticated,
    isSessionLoading: false,
    canResetDemoData: false,
    login: vi.fn(),
    logout: vi.fn(),
    user: isAuthenticated ? { email: 'test@example.com' } : null,
  });

  return render(
    <MemoryRouter initialEntries={['/admin']}>
      <Routes>
        <Route path="/admin/login" element={<div>Login Page</div>} />
        <Route
          path="/admin"
          element={
            <ProtectedRoute>
              <div data-testid="protected-content">Protected Content</div>
            </ProtectedRoute>
          }
        />
      </Routes>
    </MemoryRouter>
  );
};

describe('ProtectedRoute [REQ:ADMIN-AUTH]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[REQ:ADMIN-AUTH] should render children when authenticated', () => {
    renderWithAuth(true);

    expect(screen.getByTestId('protected-content')).toBeInTheDocument();
    expect(screen.getByText('Protected Content')).toBeInTheDocument();
  });

  it('[REQ:ADMIN-AUTH] should redirect to login when not authenticated', () => {
    renderWithAuth(false);

    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
    expect(screen.getByText('Login Page')).toBeInTheDocument();
  });

  it('should handle nested children elements', () => {
    vi.spyOn(useAdminAuthModule, 'useAdminAuth').mockReturnValue({
      isAuthenticated: true,
      isSessionLoading: false,
      canResetDemoData: false,
      login: vi.fn(),
      logout: vi.fn(),
      user: { email: 'test@example.com' },
    });

    render(
      <MemoryRouter>
        <Routes>
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <div>
                  <h1>Title</h1>
                  <p>Paragraph</p>
                </div>
              </ProtectedRoute>
            }
          />
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText('Title')).toBeInTheDocument();
    expect(screen.getByText('Paragraph')).toBeInTheDocument();
  });
});
