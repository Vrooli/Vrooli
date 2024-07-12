import { AutocompleteOption, CommonKey } from "@local/shared";
import { InputProps } from "@mui/material";
import { SxType } from "types";
import { SearchItem } from "utils/search/siteToSearch";

export type SiteSearchBarProps = Omit<InputProps, "onChange" | "onInputChange"> & {
    debounce?: number;
    /** Label for virtual keyboard's Enter/Return button */
    enterKeyHint?: string;
    id?: string;
    /** 
     * If true, we'll use a div instead of a form for the search bar component.
     * Nested forms are not allowed in HTML
     */
    isNested?: boolean;
    loading?: boolean;
    onChange: (updatedText: string) => unknown;
    onInputChange: (newValue: AutocompleteOption) => unknown;
    options?: readonly AutocompleteOption[];
    placeholder?: CommonKey;
    value: string;
    sxs?: {
        paper?: SxType;
        root?: SxType;
    };
}

export type SettingsSearchBarProps = InputProps & {
    debounce?: number;
    id?: string;
    onChange: (updatedText: string) => unknown;
    onInputChange: (newValue: SearchItem) => unknown;
    placeholder?: CommonKey;
    value: string;
}
