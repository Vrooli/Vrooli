import { AutocompleteOption, CommonKey } from "@local/shared";
import { InputProps } from "@mui/material";
import { SxType } from "types";
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
}

export type SettingsSearchBarProps = InputProps & {
    debounce?: number;
    id?: string;
    onChange: (updatedText: string) => unknown;
    onInputChange: (newValue: SearchItem) => unknown;
    placeholder?: CommonKey;
    value: string;
}
