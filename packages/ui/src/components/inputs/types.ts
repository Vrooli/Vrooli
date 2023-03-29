import { BoxProps, SwitchProps, TextFieldProps } from '@mui/material';
import { Comment, CommentFor, StandardVersion, Tag } from '@shared/consts';
import { SvgComponent } from '@shared/icons';
import { JSONVariable } from 'forms/types';
import { TagShape } from 'utils/shape/models/tag';
import { StringSchema } from 'yup';

export interface CommentCreateInputProps {
    handleClose: () => void;
    language: string;
    objectId: string;
    objectType: CommentFor;
    onCommentAdd: (comment: Comment) => any;
    parent: Comment | null;
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
    canUpdate: boolean;
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
    disabled?: boolean;
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
    index?: number;
    minRows?: number;
    name: string;
    placeholder?: string;
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
    /**
     * All languages that currently have translations for the object being edited.
     */
    languages: string[];
    zIndex: number;
}

export interface MarkdownInputProps {
    autoFocus?: boolean;
    disabled?: boolean;
    minRows?: number;
    name: string;
    placeholder?: string;
    sxs?: { bar?: { [x: string]: any }; textArea?: { [x: string]: any } };
    tabIndex?: number;
}

export type MarkdownInputBaseProps = Omit<TextFieldProps, 'onChange'> & {
    autoFocus?: boolean;
    disabled?: boolean;
    error?: boolean;
    helperText?: string | boolean | null | undefined;
    minRows?: number;
    name: string;
    onBlur?: (event: React.FocusEvent<HTMLTextAreaElement>) => void;
    onChange: (newText: string) => any;
    placeholder?: string;
    tabIndex?: number;
    value: string;
    sxs?: { bar?: { [x: string]: any }; textArea?: { [x: string]: any } };
}

export type PasswordTextFieldProps = TextFieldProps & {
    autoComplete?: string;
    autoFocus?: boolean;
    fullWidth?: boolean;
    label?: string;
    name: string;
}

export type PreviewSwitchProps = Omit<BoxProps, 'onChange'> & {
    disabled?: boolean;
    isPreviewOn: boolean;
    onChange: (isPreviewOn: boolean) => void;
}

export interface IntegerInputProps extends BoxProps {
    autoFocus?: boolean;
    disabled?: boolean;
    key?: string;
    initial?: number;
    label?: string;
    max?: number;
    min?: number;
    name: string;
    step?: number;
    tooltip?: string;
}

export interface ResourceListHorizontalInputProps {
    disabled?: boolean;
    isCreate: boolean;
    isLoading?: boolean;
    zIndex: number;
}

export type SelectorProps<T extends string | number | { [x: string]: any }> = {
    addOption?: {
        label: string;
        onSelect: () => void;
    };
    autoFocus?: boolean;
    disabled?: boolean;
    fullWidth?: boolean;
    getOptionDescription?: (option: T) => string | null | undefined;
    getOptionIcon?: (option: T) => SvgComponent;
    getOptionLabel: (option: T) => string;
    inputAriaLabel?: string;
    label?: string;
    multiple?: false;
    name: string;
    noneOption?: boolean;
    onChange?: (value: T | null) => any;
    options: T[];
    required?: boolean;
    sx?: { [x: string]: any };
    tabIndex?: number;
}

export type StandardVersionSelectSwitchProps = Omit<SwitchProps, 'onChange'> & {
    selected: {
        translations: StandardVersion['translations'];
    } | null;
    onChange: (value: StandardVersion | null) => any;
    disabled?: boolean;
    zIndex: number;
}

export interface TagSelectorProps {
    disabled?: boolean;
    placeholder?: string;
}

export interface TagSelectorBaseProps {
    disabled?: boolean;
    handleTagsUpdate: (tags: (TagShape | Tag)[]) => any;
    placeholder?: string;
    tags: (TagShape | Tag)[];
}

export interface TranslatedMarkdownInputProps {
    disabled?: boolean;
    language: string;
    minRows?: number;
    name: string;
    placeholder?: string;
    sxs?: { bar?: { [x: string]: any }; textArea?: { [x: string]: any } };
}

export interface TranslatedTextFieldProps {
    fullWidth?: boolean;
    label?: string;
    language: string;
    maxRows?: number;
    minRows?: number;
    multiline?: boolean;
    name: string;
    placeholder?: string;
}

export type VersionInputProps = Omit<TextFieldProps, 'helperText' | 'onBlur' | 'onChange' | 'value'> & {
    autoFocus?: boolean;
    fullWidth?: boolean;
    /**
     * Label for input component, NOT the version label.
     */
    label?: string;
    /**
     * Existing versions of the object. Used to determine mimum version number.
     */
    versions: string[];
}