import { InputProps, SelectProps } from '@mui/material';
import { ChangeEvent } from 'react';

export interface DropzoneProps {
    acceptedFileTypes?: string[];
    dropzoneText?: string;
    onUpload: (files: any[]) => any;
    showThumbs?: boolean;
    maxFiles?: number;
    uploadText?: string;
    cancelText?: string;
    disabled?: boolean;
}

export interface SearchBarProps extends InputProps {
    id?: string;
    placeholder?: string;
    value: string;
    onChange: (updatedText: string) => any;
    debounce?: number;
}

export interface AutocompleteSearchBarProps<T> extends SearchBarProps {
    id?: string;
    placeholder?: string;
    value: string;
    loading?: boolean;
    options?: T[];
    getOptionKey: (option: T) => string;
    getOptionLabel: (option: T) => string;
    getOptionLabelSecondary?: (option: T) => string;
    onChange: (updatedText: string) => any;
    onInputChange: (newValue: T) => any;
    debounce?: number;
}

export interface SelectorProps extends SelectProps {
    options: any[];
    getOptionLabel?: (option: any) => string;
    selected: any;
    handleChange: (change: any) => any;
    fullWidth?: boolean;
    multiple?: boolean;
    inputAriaLabel?: string;
    noneOption?: boolean;
    label?: string;
    required?: boolean;
    disabled?: boolean;
    color?: string;
    className?: string;
    style?: any;
}

export interface TagSelectorProps {
    tags: string[];
    onTagAdd: (tag: string) => any;
    onTagRemove: (tag: string) => any;
    onTagsClear: () => any;
}