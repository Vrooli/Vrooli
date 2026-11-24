export type RevisionedOperatorState = {
  updatedAt?: string;
  version?: string;
};

// Untouched operator state has an empty updatedAt and uses its schema version
// as the revision token until the first durable write.
export function operatorStateRevision(state: RevisionedOperatorState | null | undefined): string {
  return state?.updatedAt?.trim() || state?.version?.trim() || "";
}
