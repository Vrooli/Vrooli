import { createClient } from "@connectrpc/connect";
import { onboardingTransport } from "./base";
import {
  ResourcesService,
  type GetResourceHealthResponse,
  type GetResourceResponse,
  type ListDerivedResourcesResponse,
  type ListResourcesResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/resources/resources_pb";

const client = createClient(ResourcesService, onboardingTransport());

export function fetchResources(target = "local"): Promise<ListResourcesResponse> {
  return client.listResources({ target }) as unknown as Promise<ListResourcesResponse>;
}
export function fetchResource(name: string, target = "local"): Promise<GetResourceResponse> {
  return client.getResource({ target, name }) as unknown as Promise<GetResourceResponse>;
}
export function fetchResourceHealth(target = "local"): Promise<GetResourceHealthResponse> {
  return client.getResourceHealth({ target }) as unknown as Promise<GetResourceHealthResponse>;
}
export function fetchDerivedResources(target = "local"): Promise<ListDerivedResourcesResponse> {
  return client.listDerivedResources({ target }) as unknown as Promise<ListDerivedResourcesResponse>;
}
