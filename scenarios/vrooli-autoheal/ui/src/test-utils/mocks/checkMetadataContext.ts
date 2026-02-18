type CheckMetadata = {
  description?: string;
  importance?: string;
  category?: string;
};

let metadataByCheckId: Record<string, CheckMetadata> = {};

export function setMockCheckMetadata(next: Record<string, CheckMetadata>): void {
  metadataByCheckId = next;
}

export function resetMockCheckMetadata(): void {
  metadataByCheckId = {};
}

export function useMockCheckMetadata() {
  return {
    getTitle: (checkId: string) => checkId,
    getMetadata: (checkId: string) => metadataByCheckId[checkId],
    isLoading: false,
  };
}
