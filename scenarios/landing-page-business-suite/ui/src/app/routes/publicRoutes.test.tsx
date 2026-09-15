import { Suspense } from 'react';
import { MemoryRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { cleanup, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { publicRoutes } from './publicRoutes';
vi.mock('../../surfaces/public-landing/routes/PublicLanding', () => ({ PublicLanding: () => <p>Canonical page route</p> }));
vi.mock('../../surfaces/public-landing/presentation/DownloadPage', () => ({ DownloadPage: () => <p>Configured download workflow</p> }));
afterEach(cleanup);
function Location() { return <span data-testid="location">{useLocation().pathname}</span>; }
describe('public route table', () => {
  it('mounts the explicit app download workflow', async () => {
    render(<MemoryRouter initialEntries={['/apps/example/download']}><Suspense><Routes>{publicRoutes}</Routes></Suspense></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByText('Configured download workflow')).toBeInTheDocument();
  });
  it.each(['/', '/apps/example'])('mounts the canonical page for %s', async path => {
    render(<MemoryRouter initialEntries={[path]}><Suspense><Routes>{publicRoutes}</Routes></Suspense></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByText('Canonical page route')).toBeInTheDocument();
  });
  it('renders unknown routes in place even alongside the existing App catch-all redirect', () => {
    render(<MemoryRouter initialEntries={['/unknown/path']}><Location /><Routes>{publicRoutes}<Route path="*" element={<Navigate to="/" replace />} /></Routes></MemoryRouter>, { withoutRouter: true });
    expect(screen.getByRole('heading', { name: 'Page not found' })).toBeInTheDocument();
    expect(screen.getByTestId('location')).toHaveTextContent('/unknown/path');
  });
});
