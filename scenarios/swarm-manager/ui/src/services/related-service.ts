import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  RelatedService,
  type GetRelatedResponse,
} from "@vrooli/proto-types/swarm-manager/v1/api/related_pb";
import { API_BASE } from "../lib/api-client";

export type RelatedTarget =
  | { kind: "backlog"; backlogKind: string; name: string }
  | { kind: "initiative"; name: string };

export interface IRelatedService {
  getRelated(target: RelatedTarget, options: { excludeHistorical: boolean; entityKinds: string[] }): Promise<GetRelatedResponse>;
}

export function createRelatedService(): IRelatedService {
  const client = createClient(RelatedService, createConnectTransport({ baseUrl: API_BASE }));

  return {
    getRelated(target, options) {
      return client.getRelated({
        target: target.kind === "backlog"
          ? { case: "backlog", value: { kind: target.backlogKind, name: target.name } }
          : { case: "initiative", value: { name: target.name } },
        excludeHistorical: options.excludeHistorical,
        entityKinds: options.entityKinds,
      });
    },
  };
}

export const relatedService = createRelatedService();
