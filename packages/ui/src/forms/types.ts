import { InputType, NoteVersion, Reminder, ReminderCreateInput } from "@local/shared";
import { CodeInputProps as CP, DropzoneProps as DP, IntegerInputProps as QP, LanguageInputProps as LP, SelectorProps as SP, TagSelectorProps as TP } from "components/inputs/types";
import { FormikProps } from "formik";
import { MakeLazyRequest } from "hooks/useLazyFetch";
import { ReactNode } from "react";
import { Forms } from "utils/consts";
import { ApiVersionShape } from "utils/shape/models/apiVersion";
import { BookmarkListShape } from "utils/shape/models/bookmarkList";
import { BotShape } from "utils/shape/models/bot";
import { ChatShape } from "utils/shape/models/chat";
import { CommentShape } from "utils/shape/models/comment";
import { FocusModeShape } from "utils/shape/models/focusMode";
import { MeetingShape } from "utils/shape/models/meeting";
import { NodeShape } from "utils/shape/models/node";
import { NodeEndShape } from "utils/shape/models/nodeEnd";
import { NodeRoutineListShape } from "utils/shape/models/nodeRoutineList";
import { NodeRoutineListItemShape } from "utils/shape/models/nodeRoutineListItem";
import { NoteVersionShape } from "utils/shape/models/noteVersion";
import { OrganizationShape } from "utils/shape/models/organization";
import { ProfileShape } from "utils/shape/models/profile";
import { ProjectVersionShape } from "utils/shape/models/projectVersion";
import { QuestionShape } from "utils/shape/models/question";
import { ReminderShape } from "utils/shape/models/reminder";
import { ReportShape } from "utils/shape/models/report";
import { ResourceShape } from "utils/shape/models/resource";
import { RoutineVersionShape } from "utils/shape/models/routineVersion";
import { RunProjectShape } from "utils/shape/models/runProject";
import { RunRoutineShape } from "utils/shape/models/runRoutine";
import { ScheduleShape } from "utils/shape/models/schedule";
import { SmartContractVersionShape } from "utils/shape/models/smartContractVersion";
import { StandardVersionShape } from "utils/shape/models/standardVersion";
import { TagShape } from "utils/shape/models/tag";
import { ViewDisplayType } from "views/types";

//==============================================================
/* #region Specific Form Props */
//==============================================================
export interface BaseFormProps {
    children: ReactNode;
    dirty?: boolean;
    display: ViewDisplayType;
    enableReinitialize?: boolean;
    isLoading?: boolean;
    maxWidth?: number;
    onClose?: () => unknown;
    promptBeforeUnload?: boolean;
    ref?: any;
    style?: { [x: string]: string | number | null };
    validationSchema?: any;
}

export interface BaseObjectFormProps<T> extends FormikProps<T> {
    display: ViewDisplayType;
    isCreate: boolean;
    isLoading: boolean;
    isOpen: boolean;
    onCancel: () => unknown;
    onClose: () => unknown;
    ref: React.RefObject<any>;
}

export interface BaseGeneratedFormProps {
    schema: FormSchema;
    onSubmit: (values: any) => void;
}

export interface FormProps {
    onFormChange?: (form: Forms) => unknown;
}

export interface ForgotPasswordFormProps extends FormProps {
    onClose?: () => unknown;
}

export interface LogInFormProps extends FormProps {
    onClose?: () => unknown;
}

export interface ResetPasswordFormProps extends FormProps {
    onClose?: () => unknown;
}

export interface SignUpFormProps extends FormProps {
    onClose?: () => unknown;
}

export interface ApiFormProps extends BaseObjectFormProps<ApiVersionShape> {
    versions: string[];
}
export type BookmarkListFormProps = BaseObjectFormProps<BookmarkListShape>
export type BotFormProps = BaseObjectFormProps<BotShape>
export type ChatFormProps = BaseObjectFormProps<ChatShape>
export type CommentFormProps = BaseObjectFormProps<CommentShape>
export type NodeWithEndShape = NodeShape & { end: NodeEndShape };
export type NodeWithRoutineListShape = NodeShape & { routineList: NodeRoutineListShape };
export interface NodeEndFormProps extends BaseObjectFormProps<NodeWithEndShape> {
    isEditing: boolean;
}
export interface NodeRoutineListFormProps extends BaseObjectFormProps<NodeWithRoutineListShape> {
    isEditing: boolean;
}
export type FocusModeFormProps = BaseObjectFormProps<FocusModeShape>
export type MeetingFormProps = BaseObjectFormProps<MeetingShape>
export interface NoteFormProps extends BaseObjectFormProps<NoteVersionShape> {
    disabled: boolean;
    handleClose: (_?: unknown, reason?: "backdropClick" | "escapeKeyDown" | undefined) => void;
    handleDeleted: (data: NoteVersion) => void;
    versions: string[];
}
export type OrganizationFormProps = BaseObjectFormProps<OrganizationShape>
export interface ProjectFormProps extends BaseObjectFormProps<ProjectVersionShape> {
    versions: string[];
}
export type QuestionFormProps = BaseObjectFormProps<QuestionShape>
export interface ReminderFormProps extends BaseObjectFormProps<ReminderShape> {
    index?: number;
    fetchCreate: MakeLazyRequest<ReminderCreateInput, Reminder>;
    handleClose: (_?: unknown, reason?: "backdropClick" | "escapeKeyDown" | undefined) => void;
    handleCreated: (data: Reminder) => void;
    handleDeleted: (data: Reminder) => void;
    reminderListId?: string;
}
export type ReportFormProps = BaseObjectFormProps<ReportShape>
export type ResourceFormProps = BaseObjectFormProps<ResourceShape>
export interface RoutineFormProps extends BaseObjectFormProps<RoutineVersionShape> {
    isSubroutine: boolean;
    versions: string[];
}
export type RunProjectFormProps = BaseObjectFormProps<RunProjectShape>
export type RunRoutineFormProps = BaseObjectFormProps<RunRoutineShape>
export type ScheduleFormProps = BaseObjectFormProps<ScheduleShape> & {
    canSetScheduleFor: boolean;
}
export interface SmartContractFormProps extends BaseObjectFormProps<SmartContractVersionShape> {
    versions: string[];
}
export interface SubroutineFormProps extends Omit<BaseObjectFormProps<NodeRoutineListItemShape>, "display" | "isLoading"> {
    /**
     * True if the routine version itself can be updated. Otherwise, 
     * only node-level properties can be updated (e.g. index)
     */
    canUpdateRoutineVersion: boolean;
    handleViewFull: () => void;
    isEditing: boolean;
    /** Number of subroutines in parent routine list */
    numSubroutines: number;
    versions: string[];
}
export interface StandardFormProps extends BaseObjectFormProps<StandardVersionShape> {
    versions: string[];
}
export type UserFormProps = BaseObjectFormProps<ProfileShape>

//==============================================================
/* #endregion Specific Form Props */
//==============================================================

//==============================================================
/* #region Input Component Data */
//==============================================================

/**
 * Props for rendering a Checkbox input component
 */
export interface CheckboxProps {
    color?: "primary" | "secondary" | "default";
    /**
     * Array of booleans, one for each option in props
     */
    defaultValue?: boolean[];
    /**
     * Array of checkbox options
     */
    options: { label: string }[];
    /**
     * If true, displays options in a row
     */
    row?: boolean;
}

/**
 * Props for rendering a Checkbox input component
 */
export interface DropzoneProps extends Omit<DP, "onUpload" | "zIndex"> {
    defaultValue?: [];
} // onUpload handled by form

/**
 * Props for rendering a LanguageInput component
 */
export interface LanguageInputProps extends Omit<LP, "currentLanguage" | "handleAdd" | "handleChange" | "handleDelete" | "handleCurrent" | "languages" | "zIndex"> {
    defaultValue?: string[];
}

/**
 * Props for rendering a language standard input component (e.g. JSON, TypeScript, HTML, etc.)
 */
export interface CodeProps extends Omit<CP, "id" | "onChange" | "value" | "zIndex"> {
    defaultValue?: string;
}

/**
 * Props for rendering a Radio button input component
 */
export interface RadioProps {
    /** The initial value of the radio button. Must be one of the values in the `options` prop. */
    defaultValue?: string | null;
    /** Radio button options. */
    options: { label: string; value: string; }[];
    /** If true, displays options in a row. */
    row?: boolean;
}

/**
 * Props for rendering a Selector input component
 */
export interface SelectorProps<T extends string | number | { [x: string]: any; }> extends Omit<SP<T>, "selected" | "handleChange" | "zIndex"> {
    defaultValue?: any; // Ignored for now
}

/**
 * Props for rendering a Slider input component
 */
export interface SliderProps {
    min: number;
    max: number;
    defaultValue?: number; // Maps to defaultValue
    step?: number;
    valueLabelDisplay?: "auto" | "on" | "off";
}

/**
 * Props for rendering a Switch input component (aka a fancy checkbox)
 */
export interface SwitchProps {
    defaultValue?: boolean; // Maps to defaultChecked
    size?: "small" | "medium";
    color?: "primary" | "secondary" | "error" | "info" | "success" | "warning" | "default";
}

/**
 * Props for rendering a TagSelector input component
 */
export interface TagSelectorProps extends Omit<TP, "currentLanguage" | "tags" | "handleTagsUpdate" | "name" | "zIndex"> {
    defaultValue?: TagShape[];
}

/**
 * Props for rendering a text input component (the most common input component)
 */
export interface TextProps {
    /** Initial value of the text field. */
    defaultValue?: string;
    /** Autocomplete attribute for auto-filling the text field (e.g. 'username', 'current-password') */
    autoComplete?: string;
    /** If true, displays RichInput instead of TextField */
    isMarkdown?: boolean;
    /** Maximum number of characters for the text field. Defaults to 1000 */
    maxChars?: number;
    /** Maximum number of rows for the text field. Defaults to 2. */
    maxRows?: number;
    /** Minimum number of rows for the text field. Defaults to 4. */
    minRows?: number;
    placeholder?: string;
}

/**
 * Props for rendering a IntegerInput input component
 */
export interface IntegerInputProps extends Omit<QP, "defaultValue" | "name"> {
    defaultValue?: any; // Ignored, but required by some components
}

/**
 * Common props required by every FieldData type
 */
export interface FieldDataBase {
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
export interface FieldDataCheckbox extends FieldDataBase {
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
export interface FieldDataDropzone extends FieldDataBase {
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
 * Field data type and props for JSON input components
 */
export interface FieldDataJSON extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.JSON;
    /**
     * Extra props for the input component, depending on the type
     */
    props: CodeProps
}

/**
 * Field data type and props for LanguageInput input components
 */
export interface FieldDataLanguageInput extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.LanguageInput;
    /**
     * Extra props for the input component, depending on the type
     */
    props: LanguageInputProps;
}

/**
 * Field data type and props for Radio button input components
 */
export interface FieldDataRadio extends FieldDataBase {
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
export interface FieldDataSelector extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.Selector;
    /**
     * Extra props for the input component, depending on the type
     */
    props: SelectorProps<any>;
}

/**
 * Field data type and props for Slider input components
 */
export interface FieldDataSlider extends FieldDataBase {
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
export interface FieldDataSwitch extends FieldDataBase {
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
 * Field data type and props for TagSelector input components
 */
export interface FieldDataTagSelector extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.TagSelector;
    /**
     * Extra props for the input component, depending on the type
     */
    props: TagSelectorProps;
}

/**
 * Field data type and props for TextField input components
 */
export interface FieldDataText extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.Text;
    /**
     * Extra props for the input component, depending on the type
     */
    props: TextProps;
}

/**
 * Field data type and props for IntegerInput input components
 */
export interface FieldDataIntegerInput extends FieldDataBase {
    /**
     * The type of the field
     */
    type: InputType.IntegerInput;
    /**
     * Extra props for the input component, depending on the type
     */
    props: IntegerInputProps;
}

/**
 * Shape of a field's data structure. Parsed from its JSON representation
 */
export type FieldData =
    FieldDataCheckbox |
    FieldDataDropzone |
    FieldDataJSON |
    FieldDataLanguageInput |
    FieldDataIntegerInput |
    FieldDataRadio |
    FieldDataSelector |
    FieldDataSlider |
    FieldDataSwitch |
    FieldDataTagSelector |
    FieldDataText;

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
    /**
     * Total number of fields in this container
     */
    totalItems: number;
    /**
     * Determines if the border should be displayed
     */
    showBorder?: boolean;
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
    fields: FieldData[];
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
