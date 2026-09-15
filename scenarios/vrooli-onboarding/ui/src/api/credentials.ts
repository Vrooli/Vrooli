import { createClient } from "@connectrpc/connect";
import { onboardingTransport } from "./base";
import {
  CredentialsService,
  type DiagnoseCredentialsResponse,
  type ListCredentialsResponse,
  type ProvisionCredentialResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/credentials/credentials_pb";

const client = createClient(CredentialsService, onboardingTransport());

export interface CredentialListItem {
  resource: string;
  logical_id: string;
  field: string;
  label: string;
  description?: string;
  placeholder?: string;
  tiers?: string[];
  obtain_url?: string;
  required: boolean;
  provisioning?: string;
  derived_from?: string;
  status: string;
  detail?: string;
}

export interface CredentialListResponse {
  credentials: CredentialListItem[];
  count: number;
}

export function fetchCredentials(target = "local"): Promise<CredentialListResponse> {
  return (client.listCredentials({ target }) as unknown as Promise<ListCredentialsResponse>).then((response) => ({
    credentials: response.credentials.map((item) => {
      const metadata = item as typeof item & { tiers?: string[] };
      return {
        resource: item.resource,
        logical_id: item.logicalId,
        field: item.field,
        label: item.label,
        description: item.description,
        placeholder: item.placeholder,
        tiers: metadata.tiers,
        obtain_url: item.obtainUrl,
        required: item.required,
        provisioning: item.provisioning,
        derived_from: item.derivedFrom,
        status: item.status,
        detail: item.detail,
      };
    }),
    count: response.count,
  }));
}

export function provisionCredential(input: { logical_id: string; field: string; value: string }, target = "local"): Promise<ProvisionCredentialResponse> {
  return client.provisionCredential({ target, logicalId: input.logical_id, field: input.field, value: input.value }) as unknown as Promise<ProvisionCredentialResponse>;
}

export function diagnoseCredentials(target = "local"): Promise<DiagnoseCredentialsResponse> {
  return client.diagnoseCredentials({ target }) as unknown as Promise<DiagnoseCredentialsResponse>;
}
