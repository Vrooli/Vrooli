import { InputType, ListObject, OrArray } from "@local/shared";
import { CodeInputProps, DropzoneProps, IntegerInputProps, LanguageInputProps, SelectorProps, TagSelectorProps } from "components/inputs/types";
import { FormikProps } from "formik";
import { Dispatch, SetStateAction } from "react";
import { SearchPageTabOption } from "utils/search/objectToSearch";
import { TagShape } from "utils/shape/models/tag";
import { CrudProps } from "views/objects/types";
import { ViewProps } from "views/types";

export type FormProps<Model extends OrArray<ListObject>, ModelShape extends OrArray<object>> = Omit<CrudProps<Model>, "isLoading"> & FormikProps<ModelShape> & {
    disabled: boolean;
    existing: ModelShape;
    handleUpdate: Dispatch<SetStateAction<ModelShape>>;
    isReadLoading: boolean;
};

export type FormBuildViewProps = ViewProps;

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

export type FormHeaderType = FormElementBase & {
    type: "Header";
    tag: "h1" | "h2" | "h3" | "h4" | "h5" | "h6";
};

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
export interface CodeFormInputProps extends Omit<CodeInputProps, "id" | "onChange" | "value" | "zIndex"> {
    defaultValue?: string;
}
/** Type-props pair for Code input components */
export interface CodeFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.JSON;
    /** Type-specific props */
    props: CodeFormInputProps;
}

/** Dropzone-specific form input props */
export interface DropzoneFormInputProps extends Omit<DropzoneProps, "onUpload" | "zIndex"> {
    defaultValue?: [];
} // onUpload handled by form
/** Type-props pair for Dropzone input components */
export interface DropzoneFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.Checkbox;
    /** Type-specific props */
    props: DropzoneFormInputProps;
}

/** Integer-specific form input props */
export interface IntegerFormInputProps extends Omit<IntegerInputProps, | "name"> {
    defaultValue?: number;
}
/** Type-props pair for Integer input components */
export interface IntegerFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.IntegerInput;
    /** Type-specific props */
    props: IntegerFormInputProps;
}

/** LanguageInput-specific form input props */
export interface LanguageFormInputProps extends Omit<LanguageInputProps, "currentLanguage" | "handleAdd" | "handleChange" | "handleDelete" | "handleCurrent" | "languages" | "zIndex"> {
    defaultValue?: string[];
}
/** Type-props pair for Language input components */
export interface LanguageFormInput extends FormInputBase {
    /** The type of the field */
    type: InputType.LanguageInput;
    /** Type-specific props */
    props: LanguageFormInputProps;
}

export type LinkItemLimitTo = {
    type: Exclude<SearchPageTabOption, "All">;
    ownerId?: string;
    projectId?: string;
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
export interface SelectorFormInputProps<T extends SelectorFormInputOption> extends Omit<SelectorProps<T>, "selected" | "handleChange" | "zIndex"> {
    defaultValue?: T;
    getOptionValue: (option: T) => string | null | undefined;
}
/** Type-props pair for Selector input components */
export interface SelectorFormInput<T extends SelectorFormInputOption> extends FormInputBase {
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
export interface TagSelectorFormInputProps extends Omit<TagSelectorProps, "currentLanguage" | "tags" | "handleTagsUpdate" | "name" | "zIndex"> {
    defaultValue?: TagShape[];
}
/** Type-props pair for TagSelector input components */
export interface TagSelectorFormInput extends FormInputBase {
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

export type FormElement = FormHeaderType | FormInputType;


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
 * MUI spacing options. Each key represents a "breakpoint", which is set based
 * on the width of the screen.
 */
export type GridItemSpacing = number | string | {
    xs?: number | "auto";
    sm?: number | "auto";
    md?: number | "auto";
    lg?: number | "auto";
    xl?: number | "auto";
}

/**
 * MUI spacing options. Each key represents a "breakpoint", which is set based
 * on the width of the screen.
 */
export type GridSpacing = number | string | Array<number | string | null> | {
    xs?: number | "auto";
    sm?: number | "auto";
    md?: number | "auto";
    lg?: number | "auto";
    xl?: number | "auto";
}

/**
 * The layout of a specific grid container (or the entire form depending on the context)
 */
export interface GridContainerBase {
    /**
     * Title of the container
     */
    title?: string;
    /**
     * Description of the container
     */
    description?: string;
    /**
     * Spacing of the container. Overrides parent spacing
     */
    spacing?: GridSpacing;
    /**
     * Direction to display items in the container. Overrides parent spacing
     */
    direction?: "column" | "row";
    /**
     * Spacing of container's columns. Overrides spacing field
     */
    columnSpacing?: GridSpacing;
    /**
     * Spacing of container's rows. Overrides spacing field
     */
    rowSpacing?: GridSpacing;
    /**
     * Spacing of individual fields in the container
     */
    itemSpacing?: GridItemSpacing;
}

/**
 * The layout of a non-form grid container. Specifies the number of total items in the grid.
 * This is used with the schema fields to determine which inputs to render.
 */
export interface GridContainer extends GridContainerBase {
    /** True if the container is not collapsible. Defaults to false. */
    disableCollapse?: boolean;
    /** Determines if the border should be displayed */
    showBorder?: boolean;
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
     * If form is stored in database, its ID
     */
    $id?: string;
    /**
     * Contains information about the overall layout of the form
     */
    formLayout?: Omit<GridContainerBase, "rowSpacing" | "columnSpacing">;
    /**
     * Contains information about subsections of the form. Subsections 
     * can only be one level deep. If this is empty, then formLayout is 
     * used instead.
     */
    containers?: GridContainer[];
    /**
     * Defines the shape of every field in the form.
     */
    fields: FormInputType[];
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
