export interface ModelFiltersState {
  capability: string;
  language: string;
  commercialOnly: boolean;
  onlyReady: boolean;
}

export const EMPTY_MODEL_FILTERS: ModelFiltersState = {
  capability: "",
  language: "",
  commercialOnly: false,
  onlyReady: false,
};

/** Unfiltered means the registry view still reflects "what will run here". */
export function isUnfiltered(filters: ModelFiltersState): boolean {
  return (
    filters.capability === "" &&
    filters.language.trim() === "" &&
    !filters.commercialOnly &&
    !filters.onlyReady
  );
}
