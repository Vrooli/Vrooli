import { BoxProps, InputProps, SelectProps, TextFieldProps, UseSwitchProps } from '@mui/material';
import { JSONVariable } from 'forms/types';
import { ChangeEvent } from 'react';
import { ListOrganization, Session, Tag } from 'types';

export interface AutocompleteSearchBarProps<T> extends SearchBarProps {
    debounce?: number;
    getOptionKey: (option: T, languages: readonly string[]) => string;
    getOptionLabel: (option: T, languages: readonly string[]) => string;
    getOptionLabelSecondary?: (option: T) => string;
    id?: string;
    loading?: boolean;
    onChange: (updatedText: string) => any;
    onInputChange: (newValue: T) => any;
    options?: T[];
    placeholder?: string;
    session: Session;
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

export interface EditableLabelProps {
    canEdit: boolean;
    handleUpdate: (newTitle: string) => void;
    placeholder?: string;
    renderLabel: (label: string) => JSX.Element;
    sxs?: { stack?: { [x: string]: any } };
    text: string;
}

export interface JSONInputProps {
    id: string;
    description?: string;
    disabled?: boolean;
    error?: boolean;
    /**
     * JSON string representing the format that the 
     * input should follow. 
     * Any key/value that's a plain string (e.g. "name", "age") 
     * must appear in the input value exactly as shown. 
     * Any key/value that's wrapped in <> (e.g. <name>, <age>) 
     * refers to a variable.
     * Any key that starts with a ? is optional.
     * Any key that's wrapped in [] (e.g. [x]) 
     * means that additional keys can be added to the object. The value of these 
     * keys is also wrapped in [] (e.g. [string], [any], [number | string])
     * to specify the types of values allowed.
     * ex: {
     * "<721>": {
     *   "<policy_id>": {
     *     "<asset_name>": {
     *       "name": "<asset_name>",
     *       "image": "<ipfs_link>",
     *       "?mediaType": "<mime_type>",
     *       "?description": "<description>",
     *       "?files": [
     *         {
     *          "name": "<asset_name>",
     *           "mediaType": "<mime_type>",
     *           "src": "<ipfs_link>"
     *         }
     *       ],
     *       "[x]": "[any]"
     *     }
     *   },
     *   "version": "1.0"
     *  }
     * }
     */
    format?: string;
    helperText?: string | null | undefined;
    minRows?: number;
    onChange: (newText: string) => any;
    placeholder?: string;
    title?: string;
    /**
     * JSON string representing the value of the input
     */
    value: string;
    /**
     * Dictionary which describes variables (e.g. <name>, <age>) in
     * the format JSON. 
     * Each variable can have a label, helper text, a type, and accepted values.
     * ex: {
     *  "policy_id": {
     *      "label": "Policy ID",
     *      "helperText": "Human-readable name of the policy.",
     *  },
     *  "721": {
     *      "label": "721",
     *      "helperText": "The transaction_metadatum_label that describes the type of data. 
     *      721 is the number for NFTs.",
     *   }
     * }
     */
    variables?: { [x: string]: JSONVariable };
}

export interface LanguageInputProps {
    /**
     * Currently-selected language, if using this component to add/edit an object
     * with translations. Not needed if using this component to select languages 
     * for an advanced search, for example.
     */
    currentLanguage?: string;
    handleAdd: (language: string) => any;
    handleChange: (oldLanguage: string, newLanguage: string) => void;
    handleDelete: (language: string) => void;
    handleSelect: (language: string) => void;
    languages: string[];
    session: Session;
}

export interface LinkButtonProps {
    onClick: (event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => any;
    text: string;
    sxs?: {
        button?: { [x: string]: any };
        text?: { [x: string]: any };
    }
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

export interface QuantityBoxProps extends BoxProps {
    autoFocus?: boolean;
    error?: boolean;
    handleChange: (newValue: number) => any;
    helperText?: string | null | undefined;
    id: string;
    key?: string;
    initial?: number;
    label?: string;
    max?: number;
    min?: number;
    step?: number;
    tooltip?: string;
    value: number;
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

export interface ThemeSwitchProps {
    theme: 'light' | 'dark';
    onChange: (theme: 'light' | 'dark') => any;
}

export interface UserOrganizationSwitchProps extends UseSwitchProps {
    session: Session;
    selected: ListOrganization | null;
    onChange: (value: ListOrganization | null) => any;
    disabled?: boolean;
}