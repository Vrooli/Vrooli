import { GqlModelType } from "@shared/consts";

export interface AutocompleteOption {
    type: 'Loading' | 'Shortcut' | GqlModelType;
    id: string;
    label: string | null;
    [key: string]: any;
}