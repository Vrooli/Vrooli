import { BoxProps, InputProps, SelectProps, TextFieldProps, UseSwitchProps } from '@mui/material';
import { ChangeEvent, MouseEvent } from 'react';
import { Organization, Session, Tag } from 'types';

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
    getOptionKey: (option: T, languages: readonly string[]) => string;
    getOptionLabel: (option: T, languages: readonly string[]) => string;
    getOptionLabelSecondary?: (option: T) => string;
    onChange: (updatedText: string) => any;
    onInputChange: (newValue: T) => any;
    debounce?: number;
    session: Session;
}

export interface LanguageInputProps {
    currentLanguage: string;
    handleAdd: (language: string) => any;
    handleChange: (oldLanguage: string, newLanguage: string) => any;
    handleDelete: (language: string) => any;
    handleSelect: (language: string) => any;
    languages: string[];
    session: Session;
}

export interface MarkdownInputProps extends TextFieldProps {
    id: string;
    disabled?: boolean;
    error?: boolean;
    helperText?: string | null | undefined;
    onChange: (newText: string) => any;
    placeholder?: string;
    minRows?: number;
    value: string;
}

export interface QuantityBoxProps extends BoxProps {
    handleChange: (newValue: number) => any;
    id: string;
    initial?: number;
    label?: string;
    max?: number;
    min?: number;
    step?: number;
    tooltip?: string;
    value: number;
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

// Tag object which may not exist in the database
export type TagSelectorTag = Partial<Tag> & {
    tag: string
}

export interface TagSelectorProps {
    session: Session;
    tags: TagSelectorTag[];
    placeholder?: string;
    onTagAdd: (tag: TagSelectorTag) => any;
    onTagRemove: (tag: TagSelectorTag) => any;
    onTagsClear: () => any;
}

export interface UserOrganizationSwitchProps extends UseSwitchProps {
    session: Session;
    selected: Organization | null;
    onChange: (value: Organization | null) => any;
    disabled?: boolean;
}

export interface EditableLabelProps {
    canEdit: boolean;
    handleUpdate: (newTitle: string) => void;
    renderLabel: (label: string) => JSX.Element;
    sxs?: { stack?: { [x: string]: any } };
    text: string;
}