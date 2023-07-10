import { CommonKey } from "@local/shared";
import { InputProps } from "@mui/material";
import { AutocompleteOption, SxType } from "types";
import { SearchItem } from "utils/search/siteToSearch";

export type SiteSearchBarProps = Omit<InputProps, "onChange" | "onInputChange"> & {
    debounce?: number;
    id?: string;
    loading?: boolean;
    onChange: (updatedText: string) => unknown;
    onInputChange: (newValue: AutocompleteOption) => unknown;
    options?: AutocompleteOption[];
    placeholder?: CommonKey;
    showSecondaryLabel?: boolean;
    value: string;
    sxs?: {
        paper?: SxType;
        root?: SxType;
    };
    zIndex: number;
}

export type SettingsSearchBarProps = InputProps & {
    debounce?: number;
    id?: string;
    options?: SearchItem[];
    onChange: (updatedText: string) => unknown;
    onInputChange: (newValue: SearchItem) => unknown;
    placeholder?: string;
    value: string;
}
