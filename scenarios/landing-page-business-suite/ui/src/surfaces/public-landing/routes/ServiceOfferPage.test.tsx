import { screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { renderWithProviders } from '@vrooli/api-base/testing';
import { afterEach, describe, expect, it } from 'vitest';
import { ServiceOfferPage } from './ServiceOfferPage';

describe('ServiceOfferPage', () => {
  afterEach(() => document.head.querySelector('link[rel="alternate"][type="application/ld+json"]')?.remove());

  it('publishes the stable offer terms, trust limits, guest checkout, and metadata link', async () => {
    renderWithProviders(<MemoryRouter initialEntries={['/services/company-market-scan']}><ServiceOfferPage /></MemoryRouter>);

    expect(screen.getByRole('heading', { name: 'Company Market Scan' })).toBeVisible();
    expect(screen.getByText('$49.00 USD')).toBeVisible();
    expect(screen.getByText(/up to 500 characters/i)).toBeVisible();
    expect(screen.getByText(/Target delivery: 60 minutes/i)).toBeVisible();
    expect(screen.getByText(/deterministic local fixture/i)).toBeVisible();
    expect(screen.getByText(/legal review is open and operator approval is pending/i)).toBeVisible();
    expect(screen.getByRole('link', { name: 'Continue without an account' })).toHaveAttribute('href', '/service-checkout?offer_id=company-market-scan&offer_version=1');
    expect(screen.getByRole('link', { name: 'Read the free sample' })).toHaveAttribute('href', '/samples/company-market-scan.md');
    expect(screen.getByRole('link', { name: 'JSON metadata' })).toHaveAttribute('href', '/services/company-market-scan.json');
    await screen.findByRole('link', { name: 'JSON metadata' });
    expect(document.head.querySelector('link[rel="alternate"][type="application/ld+json"]')).toHaveAttribute('href', `${window.location.origin}/services/company-market-scan.json`);
  });
});
