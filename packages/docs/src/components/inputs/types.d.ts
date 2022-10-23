import { InputProps } from '@mui/material';

export interface AutocompleteSearchBarProps extends SearchBarProps {
    debounce?: number;
    id?: string;
    loading?: boolean;
    onChange: (updatedText: string) => any;
    onInputChange: (newValue: AutocompleteOption) => any;
    options?: AutocompleteOption[];
    placeholder?: string;
    showSecondaryLabel?: boolean;
    value: string;
}

export interface SearchBarProps extends InputProps {
    debounce?: number;
    id?: string;
    placeholder?: string;
    value: string;
    onChange: (updatedText: string) => any;
}

export interface ThemeSwitchProps {
    showText?: boolean;
    theme: 'light' | 'dark';
    onChange: (theme: 'light' | 'dark') => any;
}