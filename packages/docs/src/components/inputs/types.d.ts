import { BoxProps, InputProps, SelectProps, TextFieldProps, UseSwitchProps } from '@mui/material';
import { JSONVariable } from 'forms/types';
import { ChangeEvent } from 'react';
import { Organization, Session, Standard, Tag } from 'types';
import { TagShape } from 'utils';

export interface AutocompleteSearchBarProps extends SearchBarProps {
    debounce?: number;
    id?: string;
    loading?: boolean;
    onChange: (updatedText: string) => any;
    onInputChange: (newValue: AutocompleteOption) => any;
    options?: AutocompleteOption[];
    placeholder?: string;
    session: Session;
    showSecondaryLabel?: boolean;
    value: string;
}

export interface DropzoneProps {
    acceptedFileTypes?: string[];
    cancelText?: string;
    disabled?: boolean;
    dropzoneText?: string;
    maxFiles?: number;
    onUpload: (files: any[]) => any;
    showThumbs?: boolean;
    uploadText?: string;
}

export interface LanguageInputProps {
    disabled?: boolean;
    /**
     * Currently-selected language, if using this component to add/edit an object
     * with translations. Not needed if using this component to select languages 
     * for an advanced search, for example.
     */
    currentLanguage: string;
    handleAdd: (language: string) => any;
    handleDelete: (language: string) => void;
    handleCurrent: (language: string) => void;
    selectedLanguages: string[];
    session: Session;
    zIndex: number;
}

export interface MarkdownInputProps extends TextFieldProps {
    id: string;
    disabled?: boolean;
    error?: boolean;
    helperText?: string | null | undefined;
    minRows?: number;
    onChange: (newText: string) => any;
    placeholder?: string;
    value: string;
}

export interface PasswordTextFieldProps extends TextFieldProps {
    autoComplete?: string;
    autoFocus?: boolean;
    error?: boolean;
    helperText?: string | null | undefined;
    fullWidth?: boolean;
    id?: string;
    label?: string;
    name?: string;
    onBlur?: (event: FocusEventHandler<HTMLInputElement | HTMLTextAreaElement>) => void;
    onChange: (e: ChangeEvent<any>) => any;
    value: string;
}

export interface SearchBarProps extends InputProps {
    debounce?: number;
    id?: string;
    placeholder?: string;
    value: string;
    onChange: (updatedText: string) => any;
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
    disabled?: boolean;
    onTagAdd: (tag: TagShape | Tag) => any;
    onTagRemove: (tag: TagShape) => any;
    onTagsClear: () => any;
    placeholder?: string;
    session: Session;
    tags: (TagShape | Tag)[];
}

export interface ThemeSwitchProps {
    theme: 'light' | 'dark';
    onChange: (theme: 'light' | 'dark') => any;
}