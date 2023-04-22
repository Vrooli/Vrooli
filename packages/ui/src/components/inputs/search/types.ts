import { InputProps } from "@mui/material";
import { CommonKey } from "@shared/translations";
import { AutocompleteOption } from "types";
import { SearchItem } from "utils/search/siteToSearch";

export type SiteSearchBarProps = InputProps & {
    debounce?: number;
    id?: string;
    loading?: boolean;
    onChange: (updatedText: string) => any;
    onInputChange: (newValue: AutocompleteOption) => any;
    options?: AutocompleteOption[];
    placeholder?: CommonKey;
    showSecondaryLabel?: boolean;
    value: string;
    sxs?: { paper?: { [x: string]: any }, root?: { [x: string]: any } };
}

export type SettingsSearchBarProps = InputProps & {
    debounce?: number;
    id?: string;
    options?: SearchItem[];
    onChange: (updatedText: string) => any;
    onInputChange: (newValue: SearchItem) => any;
    placeholder?: string;
    value: string;
}