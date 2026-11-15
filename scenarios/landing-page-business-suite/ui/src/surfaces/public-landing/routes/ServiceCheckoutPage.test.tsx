import { fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { renderWithProviders } from '@vrooli/api-base/testing';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ServiceCheckoutPage } from './ServiceCheckoutPage';
import * as serviceOrders from '../services/serviceOrders.service';

vi.mock('../services/serviceOrders.service', () => ({
  getServiceOfferQuote: vi.fn(),
  beginServiceOrder: vi.fn(),
  verifyCaptureServiceOrder: vi.fn(),
  reconcileServiceOrder: vi.fn(),
  serviceOrderResultURL: vi.fn(() => '/api/v1/service-orders/ord_123/result'),
}));

const getQuote = vi.mocked(serviceOrders.getServiceOfferQuote);
const beginOrder = vi.mocked(serviceOrders.beginServiceOrder);
const verifyCapture = vi.mocked(serviceOrders.verifyCaptureServiceOrder);
const reconcile = vi.mocked(serviceOrders.reconcileServiceOrder);
const serviceOrderResultURL = vi.mocked(serviceOrders.serviceOrderResultURL);
const quote = { offer_id: 'company-market-scan', offer_version: 1, amount_minor: 4900, currency: 'USD' };
const authorizedOrder = { order_id: 'ord_123', job_id: 'job_123', ...quote, mode: 'test', state: 'authorized' };

function renderCheckout(path = '/service-checkout?offer_id=company-market-scan&offer_version=1') {
  return renderWithProviders(<MemoryRouter initialEntries={[path]}><ServiceCheckoutPage /></MemoryRouter>);
}

describe('ServiceCheckoutPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getQuote.mockResolvedValue(quote);
    beginOrder.mockResolvedValue(authorizedOrder);
    verifyCapture.mockResolvedValue({ ...authorizedOrder, state: 'ledger_pending', capture_id: 'cap_123' } as serviceOrders.ServiceOrder);
    reconcile.mockResolvedValue({ ...authorizedOrder, state: 'reconciled', capture_id: 'cap_123', ledger_posting_id: 'post_123', ledger_receipt_id: 'receipt_123' } as serviceOrders.ServiceOrder);
    serviceOrderResultURL.mockReturnValue('/api/v1/service-orders/ord_123/result');
  });

  it('loads canonical terms and submits only offer identity and request input', async () => {
    renderCheckout();
    expect(await screen.findByText('$49.00')).toBeVisible();
    expect(getQuote).toHaveBeenCalledWith('company-market-scan', 1);
    expect(screen.getByRole('region', { name: 'Selected test checkout channel' })).toHaveTextContent('Person in browser');
    fireEvent.change(screen.getByRole('textbox', { name: 'Market or vendor category' }), { target: { value: 'Research Example Co.' } });
    expect(screen.getByText(/I understand this creates a test-only authorization record for \$49\.00\. No funds move; a test capture is recorded only after delivery verification\./i)).toBeVisible();
    fireEvent.click(screen.getByRole('checkbox', { name: /test-only authorization record/i }));
    fireEvent.click(screen.getByRole('button', { name: 'Create $49.00 test authorization' }));
    await waitFor(() => { expect(beginOrder).toHaveBeenCalledTimes(1); });
    const body = beginOrder.mock.calls[0]![0];
    expect(body).toMatchObject({
      offer_id: 'company-market-scan', offer_version: 1, door: 'browser', job_input: { sector: 'Research Example Co.' },
    });
    expect(body.idempotency_key).toBeTruthy();
    expect(body).not.toHaveProperty('amount_minor');
    expect(await screen.findByRole('heading', { name: 'Request received' })).toBeVisible();
    expect(screen.getByText('ord_123')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: 'Verify delivery and capture test authorization' }));
    await waitFor(() => { expect(verifyCapture).toHaveBeenCalledWith('ord_123'); });
    fireEvent.click(await screen.findByRole('button', { name: 'Reconcile accepted Money Ledger receipt' }));
    await waitFor(() => { expect(reconcile).toHaveBeenCalledWith('ord_123'); });
    expect(await screen.findByRole('heading', { name: 'Test order complete' })).toBeVisible();
    expect(screen.getByText((_, element) => {
      if (!element || !element.classList.contains('site-service-checkout-receipt')) return false;
      const text = element.textContent;
      return text.includes('receipt_123') && text.includes('post_123');
    })).toBeVisible();
    expect(screen.getByRole('link', { name: 'Download your order result' })).toHaveAttribute('href', '/api/v1/service-orders/ord_123/result');
    expect(screen.getByText(/prepared sample fixture, not custom research/i)).toBeVisible();
    expect(serviceOrderResultURL).toHaveBeenCalledWith('ord_123');
  });

  it('fails closed when the canonical quote route is unavailable and offers retry', async () => {
    getQuote.mockRejectedValueOnce(new Error('Offer route unavailable')).mockResolvedValueOnce(quote);
    renderCheckout();
    expect(await screen.findByRole('heading', { name: 'Checkout unavailable' })).toBeVisible();
    expect(screen.getByText('Offer route unavailable')).toBeVisible();
    expect(beginOrder).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByText('$49.00')).toBeVisible();
  });

  it('rejects a server quote that does not match the requested offer identity', async () => {
    getQuote.mockResolvedValue({ ...quote, offer_id: 'another-offer' });
    renderCheckout();
    expect(await screen.findByText(/offer terms could not be verified/i)).toBeVisible();
    expect(screen.queryByRole('button', { name: /continue/i })).toBeNull();
  });

  it('does not call the route when offer identity is missing', async () => {
    renderCheckout('/service-checkout');
    expect(await screen.findByRole('heading', { name: 'Checkout unavailable' })).toBeVisible();
    expect(getQuote).not.toHaveBeenCalled();
  });

  it('keeps the same idempotency key when retrying the same request after a network error', async () => {
    beginOrder.mockRejectedValueOnce(new Error('Network interrupted')).mockResolvedValueOnce({ order_id: 'ord_123', job_id: 'job_123', ...quote, mode: 'test', state: 'authorized' });
    renderCheckout('/service-checkout?offer_id=company-market-scan&offer_version=1&door=browsing_agent');
    await screen.findByText('$49.00');
    expect(screen.getByRole('region', { name: 'Selected test checkout channel' })).toHaveTextContent('Browsing agent');
    expect(screen.getByText(/does not verify or assert an agent identity/i)).toBeVisible();
    fireEvent.change(screen.getByRole('textbox', { name: 'Market or vendor category' }), { target: { value: 'Research Example Co.' } });
    fireEvent.click(screen.getByRole('checkbox', { name: /test-only authorization record/i }));
    fireEvent.click(screen.getByRole('button', { name: 'Create $49.00 test authorization' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Network interrupted');
    fireEvent.click(screen.getByRole('button', { name: 'Create $49.00 test authorization' }));
    await waitFor(() => { expect(beginOrder).toHaveBeenCalledTimes(2); });
    expect(beginOrder.mock.calls[0]![0].door).toBe('browsing_agent');
    expect(beginOrder.mock.calls[1]![0].door).toBe('browsing_agent');
    expect(beginOrder.mock.calls[1]![0].idempotency_key).toBe(beginOrder.mock.calls[0]![0].idempotency_key);
  });

  it('retries verification against the same durable order after a transient failure', async () => {
    verifyCapture.mockRejectedValueOnce(new Error('Verifier temporarily unavailable'))
      .mockResolvedValueOnce({ ...authorizedOrder, state: 'ledger_pending' });
    renderCheckout();
    await screen.findByText('$49.00');
    fireEvent.change(screen.getByRole('textbox', { name: 'Market or vendor category' }), { target: { value: 'Research Example Co.' } });
    fireEvent.click(screen.getByRole('checkbox', { name: /test-only authorization record/i }));
    fireEvent.click(screen.getByRole('button', { name: 'Create $49.00 test authorization' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Verify delivery and capture test authorization' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Verifier temporarily unavailable');
    fireEvent.click(screen.getByRole('button', { name: 'Verify delivery and capture test authorization' }));
    await waitFor(() => { expect(verifyCapture).toHaveBeenCalledTimes(2); });
    expect(verifyCapture).toHaveBeenNthCalledWith(1, 'ord_123');
    expect(verifyCapture).toHaveBeenNthCalledWith(2, 'ord_123');
  });

  it('rejects an unsupported test channel before reading the offer', async () => {
    renderCheckout('/service-checkout?offer_id=company-market-scan&offer_version=1&door=agent_manager_x402');
    expect(await screen.findByRole('heading', { name: 'Checkout unavailable' })).toBeVisible();
    expect(screen.getByText(/channel is not supported/i)).toBeVisible();
    expect(screen.getByRole('region', { name: 'Selected test checkout channel' })).toHaveTextContent('Unsupported channel');
    expect(getQuote).not.toHaveBeenCalled();
  });
});
