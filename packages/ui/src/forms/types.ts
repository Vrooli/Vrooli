import { CommonProps } from "types";
import { Forms } from "utils";
import { DropzoneProps as DP, SelectorProps as SP } from 'components/inputs/types';

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
    code?: string;
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
    defaultValue?: string; // Must be one of the values in the options array
    options: { label: string; value: string }[];
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
    defaultValue?: string;
    autoComplete?: string;
    required?: boolean;
    maxRows?: number;
}

/**
 * Shape of a field's data structure. Parsed from its JSON representation
 */
export interface FieldData {
    fieldName: string;
    label: string;
    type: InputType;
    props: CheckboxProps | DropzoneProps | RadioProps | SelectorProps | SliderProps | SwitchProps | TextFieldProps;
    yup: YupField;
}

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
 * Shape of yup inside FieldData, BEFORE it is converted to an overall yup schema
 */
export interface YupField {
    required?: boolean;
    type?: string;
    checks: YupCheck[];
}

/**
 * Shape of entire Yup schema, generated from the yups in the FieldData objects
 */
export interface YupSchema {
    title: string;
    type: 'object';
    required: any[];
    properties: {[x:string]: any};
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
    title?: string; // Title of the container
    description?: string; // Description of the container
    spacing?: GridSpacing; // Spacing of the container. Overrides parent spacing
    direction?: 'column' | 'row'; // Direction to display items in the container. Overrides parent spacing
    columnSpacing?: GridSpacing; // Spacing of container's columns. Overrides spacing field
    rowSpacing?: GridSpacing; // Spacing of container's rows. Overrides spacing field
    itemSpacing?: GridItemSpacing; // Spacing of individual fields in the container
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
    $id?: string; // ID of the form in the database
    formLayout?: GridContainerBase; // Contains information about the overall layout of the form, as well as specific sections
    containers?: GridContainer[];
    fields: FieldData[];
}