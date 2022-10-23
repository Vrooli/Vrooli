import { BoxProps, InputProps, SelectProps, TextFieldProps, UseSwitchProps } from '@mui/material';
import { JSONVariable } from 'forms/types';
import { ChangeEvent } from 'react';
import { AutocompleteOption, Organization, Project, Routine, Session, Standard, Tag } from 'types';
import { ObjectType, TagShape } from 'utils';
import { StringSchema } from 'yup';

export interface AutocompleteSearchBarProps extends Omit<SearchBarProps, 'sx'> {
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
    sxs?: { paper?: { [x: string]: any }, root?: { [x: string]: any } };
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
    onDialogOpen?: (isOpen: boolean) => void;
    renderLabel: (label: string) => JSX.Element;
    sxs?: { stack?: { [x: string]: any } };
    text: string;
    validationSchema?: StringSchema<string | undefined, any, string | undefined>;
}

export interface JsonFormatInputProps {
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
    format?: { [x: string]: any };
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

export interface JsonInputProps {
    id: string;
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
    format?: { [x: string]: any }
    helperText?: string | null | undefined;
    minRows?: number;
    onChange: (newText: string) => any;
    placeholder?: string;
    /**
     * JSON string representing the value of the input
     */
    value: string | null;
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
    session: Session;
    translations: { language: string }[];
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

export interface PreviewSwitchProps extends BoxProps {
    disabled?: boolean;
    isPreviewOn: boolean;
    onChange: (isPreviewOn: boolean) => void;
}

export interface QuantityBoxProps extends BoxProps {
    autoFocus?: boolean;
    disabled?: boolean;
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

export type RelationshipItemOrganization = Pick<Organization, '__typename' | 'handle' | 'id'> & 
    { translations?: Pick<Organization['translations'][0], 'name' | 'id' | 'language'>[] };
export type RelationshipItemUser = Pick<User, '__typename' | 'handle' | 'id' | 'name'>;
export type RelationshipItemProject = Pick<Project, '__typename' | 'handle' | 'id' | 'owner'> & 
    { translations?: Pick<Project['translations'][0], 'name' | 'id' | 'language'>[] };
export type RelationshipItemRoutine = Pick<Routine, '__typename' | 'id' | 'owner'> &
    { translations?: Pick<Routine['translations'][0], 'title' | 'id' | 'language'>[] };

export type RelationshipOwner = RelationshipItemOrganization | RelationshipItemUser | null;
export type RelationshipProject = RelationshipItemProject | null;
export type RelationshipParent = RelationshipItemProject | RelationshipItemRoutine | null;

export type RelationshipsObject = {
    isComplete: boolean;
    isPrivate: boolean;
    owner: RelationshipOwner;
    parent: RelationshipParent;
    project: RelationshipProject;
}

export interface RelationshipButtonsProps {
    disabled?: boolean;
    isFormDirty?: boolean;
    objectType: ObjectType;
    onRelationshipsChange: (relationships: Partial<RelationshipsObject>) => void;
    relationships: RelationshipsObject;
    session: Session;
    zIndex: number;
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
    sx?: { [x: string]: any };
}

export interface StandardSelectSwitchProps extends UseSwitchProps {
    session: Session;
    selected: {
        name: Standard['name']
    } | null;
    onChange: (value: Standard | null) => any;
    disabled?: boolean;
    zIndex: number;
}

export interface TagSelectorProps {
    disabled?: boolean;
    handleTagsUpdate: (tags: (TagShape | Tag)[]) => any;
    placeholder?: string;
    session: Session;
    tags: (TagShape | Tag)[];
}

export interface ThemeSwitchProps {
    showText?: boolean;
    theme: 'light' | 'dark';
    onChange: (theme: 'light' | 'dark') => any;
}

export interface VersionInputProps extends TextFieldProps {
    autoFocus?: boolean;
    error?: boolean;
    helperText?: string | null | undefined;
    fullWidth?: boolean;
    id?: string;
    label?: string;
    minimum?: string;
    name?: string;
    onBlur?: (event: FocusEventHandler<HTMLInputElement | HTMLTextAreaElement>) => void;
    onChange: (newVersion: string) => any;
    value: string;
}