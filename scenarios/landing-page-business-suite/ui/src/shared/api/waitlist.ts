import { createClient } from '@connectrpc/connect';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { WaitlistService, type WaitlistEntry } from '@vrooli/proto-types/landing-page-business-suite/v1/waitlist_pb';
import { CONNECT_API_BASE } from './common';
import type { WaitlistEmail } from './types';

const waitlistClient = createClient(WaitlistService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

function entryFromProto(entry: WaitlistEntry): WaitlistEmail {
  return {
    id: Number(entry.id), email: entry.email, source: entry.source,
    created_at: entry.createdAt ? new Date(Number(entry.createdAt.seconds) * 1000 + entry.createdAt.nanos / 1_000_000).toISOString().replace('.000Z', 'Z') : '',
  };
}

export async function submitWaitlistEmail(email: string, source = 'coming_soon'): Promise<{ success: boolean; message?: string }> {
  const response = await waitlistClient.createWaitlistEntry({ email, source });
  return { success: response.success, message: response.message };
}

export async function getWaitlistEmails(): Promise<WaitlistEmail[]> {
  const response = await waitlistClient.listWaitlistEntries({});
  return response.entries.map(entryFromProto);
}

export async function deleteWaitlistEmail(id: number): Promise<{ success: boolean }> {
  const response = await waitlistClient.deleteWaitlistEntry({ id: BigInt(id) });
  return { success: response.deleted };
}

export async function exportWaitlistCsv(): Promise<{ csv: string; filename: string }> {
  const response = await waitlistClient.exportWaitlistEntries({});
  return { csv: response.csv, filename: response.filename || 'waitlist.csv' };
}
