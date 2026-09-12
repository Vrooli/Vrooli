// Snippets domain: typed Connect-RPC client and snake-case UI DTOs for
// sender-owned reusable message text.

import { Code, ConnectError, createClient } from "@connectrpc/connect";
import { SnippetsService } from "@vrooli/proto-types/web-console/v1/snippets/snippets_pb";

import { transport } from "./client";

export const snippetsClient = createClient(SnippetsService, transport);

export interface SnippetDTO {
  id: string;
  name: string;
  body: string;
  color: string;
  pinned: boolean;
  use_count: number;
  last_used_at: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

type SnippetWire = NonNullable<Awaited<ReturnType<typeof snippetsClient.upsertSnippet>>["snippet"]>;

function decodeSnippet(snippet: SnippetWire): SnippetDTO {
  return {
    id: snippet.id,
    name: snippet.name,
    body: snippet.body,
    color: snippet.color,
    pinned: snippet.pinned,
    use_count: snippet.useCount,
    last_used_at: snippet.lastUsedAt,
    sort_order: snippet.sortOrder,
    created_at: snippet.createdAt,
    updated_at: snippet.updatedAt,
  };
}

export async function listSnippets(): Promise<SnippetDTO[]> {
  const response = await snippetsClient.listSnippets({});
  return response.snippets.map(decodeSnippet);
}

export interface UpsertSnippetInput {
  id?: string;
  name: string;
  body: string;
  color?: string;
  pinned?: boolean;
  sort_order?: number;
}

export async function upsertSnippet(input: UpsertSnippetInput): Promise<SnippetDTO> {
  const response = await snippetsClient.upsertSnippet({
    id: input.id ?? "",
    name: input.name,
    body: input.body,
    color: input.color ?? "",
    pinned: input.pinned ?? false,
    hasPinned: input.pinned !== undefined,
    sortOrder: input.sort_order ?? 0,
  });
  if (!response.snippet) throw new Error("snippets.upsertSnippet: missing snippet in response");
  return decodeSnippet(response.snippet);
}

export async function deleteSnippet(id: string): Promise<boolean> {
  const response = await snippetsClient.deleteSnippet({ id });
  return response.deleted;
}

export async function touchSnippet(id: string): Promise<SnippetDTO> {
  const response = await snippetsClient.touchSnippet({ id });
  if (!response.snippet) throw new Error("snippets.touchSnippet: missing snippet in response");
  return decodeSnippet(response.snippet);
}

export async function promoteSnippet(id: string): Promise<string> {
  try {
    const response = await snippetsClient.promoteSnippet({ id });
    if (!response.identifier) {
      throw new Error("snippets.promoteSnippet: missing skill identifier in response");
    }
    return response.identifier;
  } catch (error) {
    const connectError = ConnectError.from(error);
    if (connectError.code === Code.Unavailable) {
      throw new Error("prompt-manager is not available on this host");
    }
    throw new Error(connectError.rawMessage || connectError.message);
  }
}
