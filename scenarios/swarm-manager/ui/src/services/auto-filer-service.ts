import { createClient, type Client } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AutoFilerService } from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import type { AutoFilerStatusResponse } from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import { API_BASE } from "../lib/api-client";
import type { BacklogItem, BacklogKind } from "../types";
import { mapProtoBacklogItem } from "./proto-contracts";

export type AutoFilerClient = Client<typeof AutoFilerService>;

export interface IAutoFilerService {
  getStatus(): Promise<AutoFilerStatusResponse>;
  runNow(): Promise<AutoFilerStatusResponse>;
  dismissSuggestion(kind: BacklogKind, name: string, reason?: string): Promise<BacklogItem>;
}

function createAutoFilerClient(): AutoFilerClient {
  return createClient(AutoFilerService, createConnectTransport({ baseUrl: API_BASE }));
}

export function createAutoFilerService(client: AutoFilerClient = createAutoFilerClient()): IAutoFilerService {
  return {
    async getStatus(): Promise<AutoFilerStatusResponse> {
      return client.getStatus({});
    },

    async runNow(): Promise<AutoFilerStatusResponse> {
      return client.runNow({});
    },

    async dismissSuggestion(kind: BacklogKind, name: string, reason?: string): Promise<BacklogItem> {
      const response = await client.dismissSuggestion({
        kind,
        name,
        reason: reason?.trim() || undefined,
      });
      if (!response.item) {
        throw new Error("auto-filer dismiss response missing item");
      }
      return mapProtoBacklogItem(response.item);
    },
  };
}

export const autoFilerService = createAutoFilerService();
