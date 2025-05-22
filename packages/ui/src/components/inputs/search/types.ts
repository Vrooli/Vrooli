import { type TranslationKeyCommon } from "@local/shared";
import { type InputProps } from "@mui/material";
import { type SearchItem } from "../../../utils/search/siteToSearch.js";

export type SettingsSearchBarProps = InputProps & {
    debounce?: number;
    id?: string;
    onChange: (updatedText: string) => unknown;
    onInputChange: (newValue: SearchItem) => unknown;
    placeholder?: TranslationKeyCommon;
    value: string;
}
