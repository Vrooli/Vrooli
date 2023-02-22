import { InputProps } from '@mui/material';
import { Session } from '@shared/consts';
import { CommonKey } from '@shared/translations';
import { AutocompleteOption } from 'types';
import { SearchItem } from 'utils';

export type SiteSearchBarProps = InputProps & {
    debounce?: number;
    id?: string;
    loading?: boolean;
    onChange: (updatedText: string) => any;
    onInputChange: (newValue: AutocompleteOption) => any;
    options?: AutocompleteOption[];
    placeholder?: CommonKey;
    session: Session;
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
    session: Session;
    value: string;
}