import { InputProps, SelectProps, UseSwitchProps } from '@mui/material';
import { ChangeEvent } from 'react';
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

// Tag object which may not exist in the database
export type TagSelectorTag = Partial<Tag> & {
    tag: string
}

export interface TagSelectorProps {
    session: Session;
    tags: TagSelectorTag[];
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