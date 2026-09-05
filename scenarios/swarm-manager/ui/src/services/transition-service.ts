import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { TransitionService } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";
import type { Transition } from "@vrooli/proto-types/swarm-manager/v1/domain/transition_pb";
import { API_BASE } from "../lib/api-client";

export interface ITransitionService {
  list(): Promise<readonly Transition[]>;
  start(transitionKey: string, subjectRef: string): Promise<{ executionId: string }>;
  apply(transitionKey: string, executionId: string): Promise<void>;
}

function defaultClient() {
  return createClient(TransitionService, createConnectTransport({ baseUrl: API_BASE }));
}

// Kept small so a catalog-projected action can be tested without a browser
// transport. This is deliberately not a UI transition registry.
export type TransitionClient = Pick<
  ReturnType<typeof defaultClient>,
  "listTransitions" | "startTransition" | "applyTransition"
>;

export function createTransitionService(client: TransitionClient = defaultClient()): ITransitionService {
  let catalog: readonly Transition[] | undefined;
  let catalogRequest: Promise<readonly Transition[]> | undefined;
  const list = async (): Promise<readonly Transition[]> => {
    if (catalog) return catalog;
    if (!catalogRequest) {
      catalogRequest = client.listTransitions({}).then((response) => {
        catalog = response.transitions;
        return catalog;
      }).finally(() => {
        catalogRequest = undefined;
      });
    }
    return catalogRequest;
  };
  return {
    list,
    async start(transitionKey, subjectRef) {
      const transition = (await list()).find((candidate) => candidate.key === transitionKey);
      if (!transition) throw new Error(`Transition ${transitionKey} is not declared.`);
      const response = await client.startTransition({ transitionKey, subjectRef: { subject: transition.subject, value: subjectRef } });
      return { executionId: response.executionId };
    },
    async apply(transitionKey, executionId) {
      await client.applyTransition({ transitionKey, executionId });
    },
  };
}

export const transitionService = createTransitionService();
