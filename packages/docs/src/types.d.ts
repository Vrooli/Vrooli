import { GqlModelType } from "@shared/consts";
import { Path } from "@shared/route";

export interface AutocompleteOption {
    type: 'Loading' | 'Shortcut' | GqlModelType;
    id: string;
    label: string | null;
    [key: string]: any;
}

// Miscellaneous types
export type SetLocation = (to: Path, options?: { replace?: boolean }) => void;