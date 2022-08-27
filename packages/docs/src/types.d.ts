
export interface AutocompleteOption {
    __typename: string;
    id: string;
    label: string | null;
    [key: string]: any;
}

// Miscellaneous types
export type SetLocation = (to: Path, options?: { replace?: boolean }) => void;