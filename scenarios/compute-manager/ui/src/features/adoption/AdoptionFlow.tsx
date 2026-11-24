import { useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";

import { selectors } from "../../consts/selectors";
import { AdoptionDraftForm, type AdoptionDraftFormCopy } from "./AdoptionDraftForm";
import { AdoptionStatusCard, type AdoptionStatusCardCopy } from "./AdoptionStatusCard";
import type { AdoptionDraft, AdoptionRecordSnapshot, AdoptionViewModel } from "./model";
import { toAdoptionViewModel } from "./model";

export interface AdoptionFlowCopy {
  draftForm: AdoptionDraftFormCopy;
  statusCard: AdoptionStatusCardCopy;
  createFailed: string;
  credentialStepUnavailable: string;
  credentialStepAction: string;
}

/**
 * Product flow seam: the route supplies the persisted Create/Adopt call and
 * the secure Onboarding navigation. This component never accepts a secret.
 */
export function AdoptionFlow({
  createAdoption,
  onOpenCredentialStep,
  copy,
}: {
  createAdoption: (draft: AdoptionDraft) => Promise<AdoptionRecordSnapshot>;
  onOpenCredentialStep: (credentialStepRef: string) => void;
  copy: AdoptionFlowCopy;
}) {
  const [record, setRecord] = useState<AdoptionViewModel>();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<"create_failed" | "credential_step_unavailable">();

  const submit = async (draft: AdoptionDraft) => {
    setSubmitting(true);
    setError(undefined);
    try {
      const persisted = await createAdoption(draft);
      const view = toAdoptionViewModel(persisted);
      setRecord(view);
      if (persisted.state === "awaiting_secret" && !persisted.credentialStepRef) {
        setError("credential_step_unavailable");
      }
    } catch {
      // Never echo arbitrary backend error text into UI or logs. Server errors
      // can contain values that belong only in the credential authority.
      setError("create_failed");
    } finally {
      setSubmitting(false);
    }
  };

  const continueToCredentialStep = () => {
    if (record?.state !== "awaiting_secret" || !record.credentialStepRef) {
      setError("credential_step_unavailable");
      return;
    }
    onOpenCredentialStep(record.credentialStepRef);
  };

  return (
    <section className="grid gap-space-md">
      {!record && <AdoptionDraftForm onSubmit={(draft) => void submit(draft)} copy={copy.draftForm} />}
      {submitting && <p role="status">{copy.draftForm.continueLabel}</p>}
      {error && <p role="alert">{error === "create_failed" ? copy.createFailed : copy.credentialStepUnavailable}</p>}
      {record && <>
        <AdoptionStatusCard adoption={record} copy={copy.statusCard} />
        {record.state === "awaiting_secret" && record.credentialStepRef && (
          <Button data-testid={selectors.adoption.credentialStep} type="button" onClick={continueToCredentialStep}>
            {copy.credentialStepAction}
          </Button>
        )}
      </>}
    </section>
  );
}
