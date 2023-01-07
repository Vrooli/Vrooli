import { BoxProps, InputProps, SelectChangeEvent, SelectProps, TextFieldProps, SwitchProps } from '@mui/material';
import { Comment, CommentFor, Organization, Project, ProjectVersion, Routine, RoutineVersion, Session, Standard, Tag, User } from '@shared/consts';
import { JSONVariable } from 'forms/types';
import { ChangeEvent, FocusEventHandler } from 'react';
import { AutocompleteOption } from 'types';
import { ObjectType, TagShape } from 'utils';
import { StringSchema } from 'yup';

export type AutocompleteSearchBarProps = Omit<SearchBarProps, 'sx' | 'onChange' | 'onInputChange'> & {
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

export interface CommentCreateInputProps {
    handleClose: () => void;
    language: string;
    objectId: string;
    objectType: CommentFor;
    onCommentAdd: (comment: Comment) => any;
    parent: Comment | null;
    session: Session;
    zIndex: number;
}

export interface CommentUpdateInputProps {
    comment: Comment;
    handleClose: () => void;
    language: string;
    objectId: string;
    objectType: CommentFor;
    onCommentUpdate: (comment: Comment) => any;
    parent: Comment | null;
    session: Session;
    zIndex: number;
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

export type MarkdownInputProps = Omit<TextFieldProps, 'onChange'> & {
    id: string;
    disabled?: boolean;
    error?: boolean;
    helperText?: string | null | undefined;
    minRows?: number;
    onBlur?: (event: React.FocusEvent<HTMLTextAreaElement>) => void;
    onChange: (newText: string) => any;
    placeholder?: string;
    value: string;
    sxs?: { bar?: { [x: string]: any }; textArea?: { [x: string]: any } };
}

export type PasswordTextFieldProps = TextFieldProps & {
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

export type PreviewSwitchProps = Omit<BoxProps, 'onChange'> & {
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

export type RelationshipItemOrganization = Pick<Organization, 'handle' | 'id'> &
{
    translations?: Pick<Organization['translations'][0], 'name' | 'id' | 'language'>[];
    type: 'Organization';
};
export type RelationshipItemUser = Pick<User, 'handle' | 'id' | 'name'> & {
    type: 'User';
}
export type RelationshipItemProjectVersion = Pick<ProjectVersion, 'id'> &
{
    root: Pick<Project, 'id' | 'handle' | 'owner'>;
    translations?: Pick<ProjectVersion['translations'][0], 'name' | 'id' | 'language'>[];
    type: 'ProjectVersion';
};
export type RelationshipItemRoutineVersion = Pick<RoutineVersion, 'id'> &
{
    root: Pick<Routine, 'id' | 'owner'>;
    translations?: Pick<RoutineVersion['translations'][0], 'name' | 'id' | 'language'>[];
    type: 'RoutineVersion';
};

export type RelationshipOwner = RelationshipItemOrganization | RelationshipItemUser | null;
export type RelationshipProject = RelationshipItemProjectVersion | null;
export type RelationshipParent = RelationshipItemProjectVersion | RelationshipItemRoutineVersion | null;

export type RelationshipsObject = {
    isComplete: boolean;
    isPrivate: boolean;
    owner: RelationshipOwner;
    parent: RelationshipParent;
    project: RelationshipProject;
}

export interface RelationshipButtonsProps {
    isEditing: boolean;
    isFormDirty?: boolean;
    objectType: ObjectType;
    onRelationshipsChange: (relationships: Partial<RelationshipsObject>) => void;
    relationships: RelationshipsObject;
    session: Session;
    zIndex: number;
}

export type SearchBarProps = InputProps & {
    debounce?: number;
    id?: string;
    placeholder?: string;
    value: string;
    onChange: (updatedText: string) => any;
}

export type SelectorProps<T extends string | number | { [x: string]: any }> = SelectProps & {
    color?: string;
    disabled?: boolean;
    fullWidth?: boolean;
    getOptionLabel: (option: T) => string;
    handleChange: (selected: T, event: SelectChangeEvent<any>) => any;
    inputAriaLabel?: string;
    label?: string;
    multiple?: false;
    noneOption?: boolean;
    options: T[];
    required?: boolean;
    selected: T | null | undefined;
    sx?: { [x: string]: any };
}

export type StandardSelectSwitchProps = Omit<SwitchProps, 'onChange'> & {
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

export type VersionInputProps = Omit<TextFieldProps, 'helperText' | 'onBlur' | 'onChange'> & {
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