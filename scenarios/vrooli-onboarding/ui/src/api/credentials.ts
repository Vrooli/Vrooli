import { createClient } from "@connectrpc/connect";
import { onboardingTransport } from "./base";
import {
  CredentialsService,
  type DiagnoseCredentialsResponse,
  type ListCredentialsResponse,
  type ProvisionCredentialResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/credentials/credentials_pb";

const client = createClient(CredentialsService, onboardingTransport());

export function fetchCredentials(target = "local"): Promise<ListCredentialsResponse> {
  return client.listCredentials({ target }) as unknown as Promise<ListCredentialsResponse>;
}

export function provisionCredential(input: { logical_id: string; field: string; value: string }, target = "local"): Promise<ProvisionCredentialResponse> {
  return client.provisionCredential({ target, logicalId: input.logical_id, field: input.field, value: input.value }) as unknown as Promise<ProvisionCredentialResponse>;
}

export function diagnoseCredentials(target = "local"): Promise<DiagnoseCredentialsResponse> {
  return client.diagnoseCredentials({ target }) as unknown as Promise<DiagnoseCredentialsResponse>;
}
