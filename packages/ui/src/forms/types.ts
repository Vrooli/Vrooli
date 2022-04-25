import { CommonProps } from "types";
import { Forms } from "utils";
import { DropzoneProps as DP, QuantityBoxProps, SelectorProps as SP } from 'components/inputs/types';

//==============================================================
/* #region Specific Form Props */
//==============================================================
export interface BaseFormProps {
    schema: FormSchema;
    onSubmit: (values: any) => any;
}

export interface FormProps extends Partial<CommonProps> {
    onFormChange?: (form: Forms) => any;
}

export interface LogInFormProps extends FormProps {
}

export interface ResetPasswordFormProps extends FormProps {
    userId?: string;
    code?: string;
}
//==============================================================
/* #endregion Specific Form Props */
//==============================================================

//==============================================================
/* #region Input Component Data */
//==============================================================

/**
 * The different types of input components supported
 */
export enum InputType {
    Checkbox = 'Checkbox',
    Dropzone = 'Dropzone',
    Radio = 'Radio',
    Selector = 'Selector',
    Slider = 'Slider',
    Switch = 'Switch',
    TextField = 'TextField',
    QuantityBox = 'QuantityBox',
}

/**
 * Props for rendering a Checkbox input component
 */
export interface CheckboxProps {
    color?: 'primary' | 'secondary' | 'default';
    defaultValue?: boolean;
}

/**
 * Props for rendering a Checkbox input component
 */
export interface DropzoneProps extends DP {
    defaultValue?: any; // Ignored
}

/**
 * Props for rendering a Radio button input component
 */
export interface RadioProps {
    /**
     * The initial value of the radio button. Must be one of the options in the `options` prop.
     */
    defaultValue?: any;
    /**
     * Radio button options.
     */
    options: { label: string; value: any }[];
    /**
     * If true, displays options in a row.
     */
    row?: boolean;
}

/**
 * Props for rendering a Selector input component
 */
export interface SelectorProps extends SP {
    defaultValue?: any; // Ignored
}

/**
 * Props for rendering a Slider input component
 */
export interface SliderProps {
    min: number;
    max: number;
    defaultValue?: number; // Maps to defaultValue
    step?: number;
    valueLabelDisplay?: 'auto' | 'on' | 'off';
}

/**
 * Props for rendering a Switch input component (aka a fancy checkbox)
 */
export interface SwitchProps {
    defaultValue?: boolean; // Maps to defaultChecked
    size?: 'small' | 'medium';
    color?: 'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning' | 'default';
}

/**
 * Props for rendering a TextField input component (the most common input component)
 */
export interface TextFieldProps {
    /**
     * Initial value of the text field.
     */
    defaultValue?: string;
    /**
     * Autocomplete attribute for auto-filling the text field (e.g. 'username', 'current-password')
     */
    autoComplete?: string;
    /**
     * Maximum number of rows for the text field. Defaults to 1.
     */
    maxRows?: number;
}

/**
 * Common props required by every FieldData type
 */
interface FieldDataBase {
    /**
     * The name of the field, as will be used by formik
     */
    fieldName: string;
    /**
     * The label to display for the field
     */
    label: string;
    /**
     * Defines validation schema for the field
     */
    yup?: YupField;
}

/**
 * Field data type and props for Checkbox input components
 */
interface FieldDataCheckbox extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.Checkbox;
    /**
     * Extra props for the input component, depending on the type
     */
    props: CheckboxProps
}


/**
 * Field data type and props for Dropzone input components
 */
interface FieldDataDropzone extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.Dropzone;
    /**
     * Extra props for the input component, depending on the type
     */
    props: DropzoneProps;
}

/**
 * Field data type and props for Radio button input components
 */
interface FieldDataRadio extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.Radio;
    /**
     * Extra props for the input component, depending on the type
     */
    props: RadioProps;
}

/**
 * Field data type and props for Selector input components
 */
interface FieldDataSelector extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.Selector;
    /**
     * Extra props for the input component, depending on the type
     */
    props: SelectorProps;
}

/**
 * Field data type and props for Slider input components
 */
interface FieldDataSlider extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.Slider;
    /**
     * Extra props for the input component, depending on the type
     */
    props: SliderProps;
}

/**
 * Field data type and props for Switch input components
 */
interface FieldDataSwitch extends FieldDataBase {
    /**
     * The type of the field
     */
     type: InputType.Switch;
     /**
      * Extra props for the input component, depending on the type
      */
     props: SwitchProps;
}

/**
 * Field data type and props for TextField input components
 */
interface FieldDataTextField extends FieldDataBase {
    /**
     * The type of the field
     */
     type: InputType.TextField;
     /**
      * Extra props for the input component, depending on the type
      */
     props: TextFieldProps;
}

/**
 * Field data type and props for QuantityBox input components
 */
interface FieldDataQuantityBox extends FieldDataBase {
    /**
     * The type of the field
     */
     type: InputType.QuantityBox;
     /**
      * Extra props for the input component, depending on the type
      */
     props: Omit<QuantityBoxProps, 'id' | 'value' | 'handleChange'>;
}

/**
 * Shape of a field's data structure. Parsed from its JSON representation
 */
export type FieldData = FieldDataCheckbox | FieldDataDropzone | FieldDataRadio | FieldDataSelector | FieldDataSlider | FieldDataSwitch | FieldDataTextField | FieldDataQuantityBox;

//==============================================================
/* #endregion Input Component Data */
//==============================================================

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
    type: 'object';
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
    xs?: number | 'auto' | boolean;
    sm?: number | 'auto' | boolean;
    md?: number | 'auto' | boolean;
    lg?: number | 'auto' | boolean;
    xl?: number | 'auto' | boolean;
}

/**
 * MUI spacing options. Each key represents a "breakpoint", which is set based
 * on the width of the screen.
 */
export type GridSpacing = number | string | Array<number | string | null> | {
    xs?: number | 'auto';
    sm?: number | 'auto';
    md?: number | 'auto';
    lg?: number | 'auto';
    xl?: number | 'auto';
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
    direction?: 'column' | 'row';
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
    totalItems: number; // Total number of fields in this container
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
    formLayout?: GridContainerBase;
    /**
     * Contains information about subsections of the form. Subsections 
     * can only be one level deep. If this is empty, then formLayout is 
     * used instead.
     */
    containers?: GridContainer[];
    /**
     * Defines the shape of every field in the form.
     */
    fields: FieldData[];
}