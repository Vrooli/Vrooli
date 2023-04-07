//==============================================================
/* #region Database Enums  */
//==============================================================
// These enums need additional typing information to be compatible with Prisma.
// For more information, see: https://github.com/prisma/prisma/discussions/9215
//==============================================================

/**
 * The different types of input components supported for forms and standards. 
 * If more specific types are needed (e.g. URLs, email addresses, etc.), these 
 * are set using Yup validation checks.
 */
export enum InputType {
    Checkbox = 'Checkbox',
    Dropzone = 'Dropzone',
    JSON = 'JSON',
    IntegerInput = 'IntegerInput',
    LanguageInput = 'LanguageInput',
    Markdown = 'Markdown',
    Radio = 'Radio',
    Selector = 'Selector',
    Slider = 'Slider',
    Switch = 'Switch',
    TagSelector = 'TagSelector',
    TextField = 'TextField',
}

export enum ViewFor {
    Api = "Api",
    ApiVersion = "ApiVersion",
    Note = "Note",
    NoteVersion = "NoteVersion",
    Organization = "Organization",
    Project = "Project",
    ProjectVersion = "ProjectVersion",
    Question = "Question",
    Routine = "Routine",
    RoutineVersion = "RoutineVersion",
    SmartContract = "SmartContract",
    SmartContractVersion = "SmartContractVersion",
    Standard = "Standard",
    StandardVersion = "StandardVersion",
    User = "User",
}

//==============================================================
/* #endregion Database Enums*/
//==============================================================