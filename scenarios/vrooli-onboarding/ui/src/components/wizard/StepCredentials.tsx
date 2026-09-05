import { StepCredentials as CredentialsSurface } from "./StepReadiness";

/** Credential entry is a dedicated wizard screen with secret-safe delivery. */
export function StepCredentials({ target = "local" }: { target?: string }) {
  return <CredentialsSurface target={target} />;
}
