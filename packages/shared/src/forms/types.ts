import { InputType, TagShape, TimeFrame } from "@local/shared";
import { SearchPageTabOption } from "../consts/search";
import { CodeLanguage } from "../consts/ui";

/**
 * Non-input form elements
 */
export enum FormStructureType {
    Divider = "Divider",
    Header = "Header",
}

export type FormBuildViewProps = {
    /** 
     * Prefix for the field names, in case you need to store multiple 
     * element sets in one formik (e.g. both inputs and outputs) 
     */
    fieldNamePrefix?: string;
    limits?: {
        inputs?: {
            types?: readonly InputType[];
        },
        structures?: {
            types: readonly FormStructureType[];
        }
    }
    onSchemaChange: (schema: FormSchema) => unknown;
    schema: FormSchema;
};

export type FormRunViewProps = {
    disabled: boolean;
    /** 
    * Prefix for the field names, in case you need to store multiple 
    * element sets in one formik (e.g. both inputs and outputs) 
    */
    fieldNamePrefix?: string;
    schema: FormSchema;
};

export type FormViewProps = FormBuildViewProps & FormRunViewProps & {
    isEditing: boolean;
};

/** Common props required by every FormElement type */
export interface FormElementBase {
    /**
    * Optional description to use as a placeholder or short description of the field.
    */
    description?: string | null;
    /**
     * Longer help text to display next to the field's label. 
     * Supports markdown.
     */
    helpText?: string | null;
    /**
     * ID for field
     */
    id: string;
    /**
     * The label to display for the field
     */
    label: string;
}

export type HeaderTag = "h1" | "h2" | "h3" | "h4" | "h5" | "h6";
export type FormHeaderType = FormElementBase & {
    type: FormStructureType.Header;
    /** 
     * Allows header to be collapsible, which hides everything under it, 
     * until the next header, page break/divider, or the end of the form.
     */
    isCollapsible?: boolean;
    tag: HeaderTag;
};

export type FormDividerType = FormElementBase & {
    type: FormStructureType.Divider;
}

/** Common props required by every form input type */
export interface FormInputBase extends FormElementBase {
    /** The name of the field, as will be used by formik */
    fieldName: string;
    /** If true, the field must be filled out before submitting the form. Defaults to false. */
    isRequired?: boolean;
    /** Validation schema for the field */
    yup?: YupField;
}

/** Checkbox-specific form input props */
export interface CheckboxFormInputProps {
    color?: "primary" | "secondary" | "default";
    /** Array of booleans. One for each option in props */
    defaultValue?: boolean[];
    /** Array of checkbox options */
    options: {
        label: string;
        value: string;
    }[];
    /**
     * The maximum number of checkboxes that can be selected. 
     * If 0 or not set, there is no limit.
     */
    maxSelection?: number;
    /**
     * The minimum number of checkboxes that must be selected.
     * If 0 or not set, there is no minimum.
     */
    minSelection?: number;
    /** If true, displays options in a row instead of a column */
    row?: boolean;
}
/** Type-props pair for Checkbox input components */
export interface CheckboxFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.Checkbox;
    /** Type-specific props */
    props: CheckboxFormInputProps;
}

/** Code-specific form input props */
export type CodeFormInputProps = {
    defaultValue?: string;
    disabled?: boolean;
    /** Limit the languages that can be selected in the language dropdown. */
    limitTo?: readonly CodeLanguage[];
}
/** Type-props pair for Code input components */
export interface CodeFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.JSON;
    /** Type-specific props */
    props: CodeFormInputProps;
}

/** Dropzone-specific form input props */
export type DropzoneFormInputProps = {
    acceptedFileTypes?: string[];
    cancelText?: string;
    defaultValue?: [];
    disabled?: boolean;
    dropzoneText?: string;
    maxFiles?: number;
    showThumbs?: boolean;
    uploadText?: string;
}
/** Type-props pair for Dropzone input components */
export type DropzoneFormInput = FormInputBase & {
    /** The type of the field */
    type: InputType.Checkbox;
    /** Type-specific props */
    props: DropzoneFormInputProps;
}

/** Integer-specific form input props */
export type IntegerFormInputProps = {
    allowDecimal?: boolean;
    autoFocus?: boolean;
    defaultValue?: number;
    disabled?: boolean;
    error?: boolean;
    fullWidth?: boolean;
    helperText?: string | boolean | null | undefined;
    key?: string;
    initial?: number;
    label?: string;
    max?: number;
    min?: number;
    offset?: number;
    step?: number;
    tooltip?: string;
    /** If provided, displays this text instead of 0 */
    zeroText?: string;
}
/** Type-props pair for Integer input components */
export type IntegerFormInput = FormInputBase & {
    /** The type of the field */
    type: InputType.IntegerInput;
    /** Type-specific props */
    props: IntegerFormInputProps;
}

/** LanguageInput-specific form input props */
export type LanguageFormInputProps = {
    defaultValue?: string[];
    disabled?: boolean;
}
/** Type-props pair for Language input components */
export type LanguageFormInput = FormInputBase & {
    /** The type of the field */
    type: InputType.LanguageInput;
    /** Type-specific props */
    props: LanguageFormInputProps;
}

export type LinkItemType = Exclude<SearchPageTabOption, "All">;
export type LinkItemLimitTo = {
    type: LinkItemType;
    advancedSearchParams?: {
        locked?: boolean;
        /** Url-encoded representation of advanced search params object */
        value: string;
    }
    ownerId?: {
        locked?: boolean;
        value: string;
    }
    projectId?: {
        locked?: boolean;
        value: string;
    }
    searchString?: {
        locked?: boolean;
        value: string;
    };
    sortBy?: {
        locked?: boolean;
        value: string;
    };
    timeFrame?: {
        locked?: boolean;
        value: TimeFrame
    };
    variant: "any" | "root" | "version";
}
/** Link item-specific form input props */
export interface LinkItemFormInputProps {
    defaultValue?: string;
    limitTo?: LinkItemLimitTo[];
    placeholder?: string;
}
/** Type-props pair for Link item input components */
export interface LinkItemFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.LinkItem;
    /** Type-specific props */
    props: LinkItemFormInputProps;
}

/** Link url-specific form input props */
export interface LinkUrlFormInputProps {
    acceptedHosts?: string[];
    defaultValue?: string;
    placeholder?: string;
}
/** Type-props pair for Link url input components */
export interface LinkUrlFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.LinkUrl;
    /** Type-specific props */
    props: LinkUrlFormInputProps;
}

/** Radio-specific form input props */
export interface RadioFormInputProps {
    /** The initial value of the radio button. Must be one of the values in the `options` prop. */
    defaultValue?: string | null;
    /** Radio button options. */
    options: { label: string; value: string; }[];
    /** If true, displays options in a row. */
    row?: boolean;
}
/** Type-props pair for Radio input components */
export interface RadioFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.Radio;
    /** Type-specific props */
    props: RadioFormInputProps;
}

export type SelectorFormInputOption = {
    description?: string;
    label: string;
    value: string;
};
/** Selector-specific form input props */
export type SelectorFormInputProps<T extends SelectorFormInputOption> = {
    addOption?: {
        label: string;
        onSelect: () => unknown;
    };
    autoFocus?: boolean;
    defaultValue?: T;
    disabled?: boolean;
    fullWidth?: boolean;
    getOptionDescription: (option: T) => string | null | undefined;
    getOptionLabel: (option: T) => string | null | undefined;
    getOptionValue: (option: T) => string | null | undefined;
    inputAriaLabel?: string;
    isRequired?: boolean,
    label?: string;
    multiple?: false;
    noneOption?: boolean;
    noneText?: string;
    options: readonly T[];
    tabIndex?: number;
}
/** Type-props pair for Selector input components */
export type SelectorFormInput<T extends SelectorFormInputOption> = FormInputBase & {
    /** The type of the field */
    type: InputType.Selector;
    /** Type-specific props */
    props: SelectorFormInputProps<T>;
}

/** Slider-specific form input props */
export interface SliderFormInputProps {
    min?: number;
    max?: number;
    defaultValue?: number;
    step?: number;
    valueLabelDisplay?: "auto" | "on" | "off";
}
/** Type-props pair for Slider input components */
export interface SliderFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.Slider;
    /** Type-specific props */
    props: SliderFormInputProps;
}

/** Switch-specific form input props */
export interface SwitchFormInputProps {
    defaultValue?: boolean;
    label?: string;
    size?: "small" | "medium";
    color?: string | "primary" | "secondary" | "error" | "info" | "success" | "warning" | "default";
}
/** Type-props pair for Switch input components */
export interface SwitchFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.Switch;
    /** Type-specific props */
    props: SwitchFormInputProps;
}

/** Tag-specific form input props */
export type TagSelectorFormInputProps = {
    defaultValue?: TagShape[];
    disabled?: boolean;
    isRequired?: boolean;
    placeholder?: string;
}
/** Type-props pair for TagSelector input components */
export type TagSelectorFormInput = FormInputBase & {
    /** The type of the field */
    type: InputType.TagSelector;
    /** Type-specific props */
    props: TagSelectorFormInputProps;
}

/** Text-specific form input props */
export interface TextFormInputProps {
    /** Initial value of the text field. */
    defaultValue?: string;
    /** Autocomplete attribute for auto-filling the text field (e.g. 'username', 'current-password') */
    autoComplete?: string;
    /** If true, displays RichInput instead of TextInput */
    isMarkdown?: boolean;
    /** Maximum number of characters for the text field. Defaults to 1000 */
    maxChars?: number;
    /** Maximum number of rows for the text field. Defaults to 2. */
    maxRows?: number;
    /** Minimum number of rows for the text field. Defaults to 4. */
    minRows?: number;
    placeholder?: string;
}
/** Type-props pair for Text input components */
export interface TextFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.Text;
    /** Type-specific props */
    props: TextFormInputProps;
}

/**
 * Shape of a field's data structure. Parsed from its JSON representation
 */
export type FormInputType =
    CheckboxFormInput |
    CodeFormInput |
    DropzoneFormInput |
    IntegerFormInput |
    LanguageFormInput |
    LinkItemFormInput |
    LinkUrlFormInput |
    RadioFormInput |
    SelectorFormInput<any> |
    SliderFormInput |
    SwitchFormInput |
    TagSelectorFormInput |
    TextFormInput;

export type FormElement = FormHeaderType | FormDividerType | FormInputType;


//==============================================================
/* #region Yup Conversion Types */
//==============================================================
// Describes the shape of the "yup" field in the FieldData object. 
// This field allows an input to use Yup validation rules
//==============================================================

// type YupDate = {
//     maxDate?: Date;
//     minDate?: Date;
// }

// type YupNumber = {
//     integer?: boolean;
//     moreThan?: number;
//     lessThan?: number;
//     positive?: boolean;
//     negative?: boolean;
//     min?: number;
//     max?: number;
//     truncate?: boolean;
//     round?: boolean;
// }

// type YupObject = {

// }

// type YupString = {
//     minLength?: number;
//     maxLength?: number;
//     pattern?: string;
//     email?: boolean;
//     url?: boolean;
//     lowercase?: boolean;
//     uppercase?: boolean;
//     trim?: boolean;
//     enum?: string[];
//     enum_titles?: string[];
// }

/**
 * Constraint applied to a yup field
 */
export interface YupCheck {
    key: string;
    value: string | number | boolean;
    error: string;
}

/**
 * Types supported by yup validation. Does not include "array" because this type is applied to 
 * a single field.
 */
export type YupType = "boolean" | "date" | "mixed" | "number" | "object" | "string";

/**
 * Shape of yup inside FieldData, BEFORE it is converted to an overall yup schema
 */
export interface YupField {
    /**
     * Indicates if the field is required
     */
    required?: boolean;
    /**
     * Specifies type of object being validated.
     */
    type?: YupType;
    /**
     * Constraints for the field (e.g. minLength, maxLength, email, integer, etc.). 
     * For example, if the type is "string", then the constraints must be functions 
     * in yup.string().
     */
    checks: YupCheck[];
}

/**
 * Shape of entire Yup schema, generated from the yups in the FieldData objects
 */
export interface YupSchema {
    title: string;
    type: "object";
    required: any[];
    properties: { [x: string]: any };
}

//==============================================================
/* #endregion Yup Conversion Types */
//==============================================================

//==============================================================
/* #region Layout Types */
//==============================================================

/**
 * The layout of a specific grid container (or the entire form depending on the context)
 */
export type GridContainerBase = {
    /**
     * Title of the container
     */
    title?: string;
    /**
     * Description of the container
     */
    description?: string;
    /**
     * Direction to display items in the container. Overrides parent spacing
     */
    direction?: "column" | "row";
}

/**
 * The layout of a non-form grid container. Specifies the number of total items in the grid.
 * This is used with the schema fields to determine which inputs to render.
 */
export type GridContainer = GridContainerBase & {
    /** True if the container is not collapsible. Defaults to false. */
    disableCollapse?: boolean;
    /** Total number of fields in this container */
    totalItems: number;
}

//==============================================================
/* #endregion Layout Types */
//==============================================================

/**
 * Schema object that specifies the following properties of a form:
 * - Overall Layout
 * - Grid Containers
 * - Input Components
 * - Titles/Descriptions of overall form, containers, and inputs
 * - Validation and Error Messages
 */
export interface FormSchema {
    /**
     * Contains information about the overall layout of the form
     */
    layout?: Omit<GridContainerBase, "direction">;
    /**
     * Contains information about subsections of the form. Subsections 
     * can only be one level deep. If this is empty, then all elements
     * will be placed in one container.
     */
    containers: GridContainer[];
    /**
     * Defines the shape of every element in the form, including headers, dividers, and inputs
     */
    elements: FormElement[];
}

/**
 * JSON-Schema format, but with keys/values that can be variables.
 * See https://json-schema.org/ for the full specification (minus the variables part).
 */
export interface JSONSchemaFormat {
    /**
     * Specifies format this schema is in. (e.g. "https://json-schema.org/draft/2020-12/schema")
     */
    $schema?: string;
    /**
     * ID of the schema, as it appears in the database.
     */
    $id?: string;
    /**
     * The title of the schema.
     */
    title?: string;
    /**
     * The description of the schema.
     */
    description?: string;
    /**
     * Specifies the type of this schema. (e.g. "object"). 
     * See https://json-schema.org/draft/2020-12/json-schema-validation.html#rfc.section.6.1.1
     */
    type?: "null" | "boolean" | "object" | "array" | "number" | "string" | "integer";
    /**
     * Specifies the properties of this schema, if type is "object".
     */
    properties?: { [x: string]: any }; // TODO
}

/**
 * Shape of the JSON input's variable definitions
 */
export interface JSONVariable {
    /**
     * The label to display for the field
     */
    label?: string;
    /**
     * Describes what the field is for, or any other relevant information
     */
    helperText?: string;
    /**
     * Defines validation schema for the field
     */
    yup?: YupField;
    /**
     * Default value. If variable is used as a key, this 
     * can only be a string, otherwise, it can be a string or valid JSON
     */
    defaultValue?: string | object;
    /**
     * Value of the field, or list of accepted values
     * //TODO might replace with yup validation
     */
    value?: string | string[];
}
