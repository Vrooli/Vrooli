import { TranslationKeyCommon } from "@local/shared";
import { InputProps } from "@mui/material";
import { SearchItem } from "../../../utils/search/siteToSearch.js";

export type SettingsSearchBarProps = InputProps & {
    debounce?: number;
    id?: string;
    onChange: (updatedText: string) => unknown;
    onInputChange: (newValue: SearchItem) => unknown;
    placeholder?: TranslationKeyCommon;
    value: string;
}
