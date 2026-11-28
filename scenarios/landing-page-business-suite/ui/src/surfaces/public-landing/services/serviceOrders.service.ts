import { buildApiUrl } from '@vrooli/api-base';
import { API_BASE, apiCall } from '../../../shared/api/common';

export interface ServiceOfferQuote {
  offer_id: string;
  offer_version: number;
  amount_minor: number;
  currency: string;
}

export interface ServiceOrder {
  order_id: string;
  job_id: string;
  offer_id: string;
  offer_version: number;
  amount_minor: number;
  currency: string;
  mode: string;
  state: string;
  ledger_posting_id?: string;
  ledger_receipt_id?: string;
}

export type ServiceOrderDoor = 'browser' | 'browsing_agent';

export async function getServiceOfferQuote(offerID: string, offerVersion: number): Promise<ServiceOfferQuote> {
  const query = new URLSearchParams({ offer_id: offerID, offer_version: String(offerVersion) });
  return apiCall<ServiceOfferQuote>(`/service-orders/offer?${query.toString()}`);
}

export function beginServiceOrder(input: {
  offer_id: string;
  offer_version: number;
  door: ServiceOrderDoor;
  idempotency_key: string;
  job_input: { sector: string };
}): Promise<ServiceOrder> {
  return apiCall<ServiceOrder>('/service-orders', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function verifyCaptureServiceOrder(orderID: string): Promise<ServiceOrder> {
  return apiCall<ServiceOrder>(`/service-orders/${encodeURIComponent(orderID)}/verify-capture`, { method: 'POST' });
}

export function reconcileServiceOrder(orderID: string): Promise<ServiceOrder> {
  return apiCall<ServiceOrder>(`/service-orders/${encodeURIComponent(orderID)}/reconcile`, { method: 'POST' });
}

export function serviceOrderResultURL(orderID: string): string {
  return buildApiUrl(`/service-orders/${encodeURIComponent(orderID)}/result`, { baseUrl: API_BASE });
}
