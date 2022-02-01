/**
 * Validation is handled by Yup. This is used by both the server and the client.
 * The server uses validation because API calls cannot be trusted to be valid.
 * The client uses validation to display errors to the user, and to simplify forms 
 * by using formik.
 * 
 * There are 5 different types of validation:
 * - connect - Associates two objects together. The child must already exist.
 * - disconnect - Removes an association between two objects. The parent and child must already exist.
 * - delete - Deletes the child object. The backend verifies whether this is allowed.
 * - add - Creates a new object.
 * - update - Updates an existing object.
 * Connect, disconnect, and delete are accomplished through passing IDs. Add and update, on the other hand, 
 * are accomplished through passing objects. 
 * Relationships in an "add" object can only be of the type connect or add. 
 * Relationships in an "update" object can be of any type.
 * 
 * Each relationship in an object can have up to 5 fields, each for the type of validation. 
 * If the relationship is "input", for example, an update object could have the fields "inputConnect", "inputDisconnect",
 * "inputAdd", "inputUpdate", and "inputDelete". 
 */
import * as yup from 'yup';
import { AccountStatus, NodeType, ResourceFor, StandardType } from './consts';

export const MIN_USERNAME_LENGTH = 3;
export const MAX_USERNAME_LENGTH = 25;
export const USERNAME_REGEX = /^[a-zA-Z0-9_.-]*$/;
export const USERNAME_REGEX_ERROR = `Characters allowed: letters, numbers, '-', '.', '_'`;
export const MIN_PASSWORD_LENGTH = 8;
export const MAX_PASSWORD_LENGTH = 50;
// See https://stackoverflow. com/a/21456918/10240279 for more options
export const PASSWORD_REGEX = /^(?=.*[A-Za-z])(?=.*\d)[A-Za-z\d@$!%*#?&]{8,}$/;
export const PASSWORD_REGEX_ERROR = "Must be at least 8 characters, with at least one character and one number";

//==============================================================
/* #region Shared fields */
//==============================================================

export const usernameSchema = yup.string().min(MIN_USERNAME_LENGTH).max(MAX_USERNAME_LENGTH).matches(USERNAME_REGEX, USERNAME_REGEX_ERROR);
export const passwordSchema = yup.string().min(MIN_PASSWORD_LENGTH).max(MAX_PASSWORD_LENGTH).matches(PASSWORD_REGEX, PASSWORD_REGEX_ERROR);
const idField = yup.string().max(256).optional();

const bioField = yup.string().max(2048).optional();
const descriptionField = yup.string().max(2048).optional();
const nameField = yup.string().max(128).optional();
const titleField = yup.string().max(128).optional();
const versionField = yup.string().max(16).optional();

const anonymousField = yup.boolean().optional();

const idArray = yup.array().of(idField.required()).optional();

//==============================================================
/* #endregion Shared fields */
//==============================================================

export const emailSchema = yup.object().shape({
    emailAddress: yup.string().max(128).required(),
    receivesAccountUpdates: yup.bool().default(true).optional(),
    receivesBusinessUpdates: yup.bool().default(true).optional(),
    userId: yup.string().optional(),
});

/**
 * Schema for submitting site feedback
 */
export const feedbackSchema = yup.object().shape({
    text: yup.string().max(4096).required(),
    userId: yup.string().optional(),
});

export const roleSchema = yup.object().shape({
    title: yup.string().max(128).required(),
    description: yup.string().max(2048).optional(),
    userIds: yup.array().of(yup.string().required()).optional(),
});

export const userSchema = yup.object().shape({
    id: yup.string().max(256).optional(),
    username: yup.string().max(128).required(),
    emails: yup.array().of(emailSchema).required(),
    status: yup.mixed().oneOf(Object.values(AccountStatus)).optional(),
});


// Schema for creating a new account
export const emailSignUpSchema = yup.object().shape({
    username: usernameSchema.required(),
    email: yup.string().email().required(),
    marketingEmails: yup.boolean().required(),
    password: passwordSchema.required(),
    passwordConfirmation: yup.string().oneOf([yup.ref('password'), null], 'Passwords must match')
});

// Schema for updating a user profile
export const profileSchema = yup.object().shape({
    username: usernameSchema.required(),
    email: yup.string().email().required(),
    theme: yup.string().max(128).required(),
    // Don't apply validation to current password. If you change password requirements, users would be unable to change their password
    currentPassword: yup.string().max(128).required(),
    newPassword: passwordSchema.optional(),
    newPasswordConfirmation: yup.string().oneOf([yup.ref('newPassword'), null], 'Passwords must match')
});

/**
 * Schema for traditional email/password log in.
 * NOTE: Does not include verification code, since it is optional and
 * the schema is reused for the log in form.
 */
export const emailLogInSchema = yup.object().shape({
    email: yup.string().email().required(),
    password: yup.string().max(128).required()
})

/**
 * Schema for sending a password reset request
 */
export const emailRequestPasswordChangeSchema = yup.object().shape({
    email: yup.string().email().required()
})

/**
 * Schema for resetting your password
 */
export const emailResetPasswordSchema = yup.object().shape({
    newPassword: passwordSchema.required(),
    confirmNewPassword: yup.string().oneOf([yup.ref('newPassword'), null], 'Passwords must match')
})

/**
 * Schema for creating a project
 */
export const addProjectSchema = yup.object().shape({
    name: yup.string().max(128).required(),
    description: yup.string().max(2048).optional(),
})

//==============================================================
/* #region Resource */
//==============================================================

const resourceLinkField = yup.string().max(1024).optional();
const resourceUsedForField = yup.string().oneOf(Object.values(ResourceFor)).optional();

/**
 * Information required when creating a resource
 */
export const resourceAddSchema = yup.object().shape({
    description: descriptionField,
    link: resourceLinkField.required(),
    title: titleField,
    usedFor: resourceUsedForField,
})

/**
 * Information required when updating a resource
 */
export const resourceUpdateSchema = yup.object().shape({
    id: idField.required(),
    description: descriptionField,
    link: resourceLinkField,
    title: titleField,
    usedFor: resourceUsedForField,
})

const resourcesConnect = idArray;
const resourcesDisconnect = idArray;
const resourcesDelete = idArray;
const resourcesAdd = yup.array().of(resourceAddSchema.required()).optional();
const resourcesUpdate = yup.array().of(resourceUpdateSchema.required()).optional();

//==============================================================
/* #endregion Resource */
//==============================================================

//==============================================================
/* #region Tag */
//==============================================================

const tagField = yup.string().max(128).optional();

/**
 * Information required when creating a resource
 */
export const tagAddSchema = yup.object().shape({
    anonymous: anonymousField, // Determines if the user will be credited for the tag
    description: descriptionField,
    tag: tagField.required(),
})

/**
 * Information required when updating a resource
 */
export const tagUpdateSchema = yup.object().shape({
    id: idField.required(),
    anonymous: anonymousField,
    description: descriptionField,
    tag: tagField,
})

const tagsConnect = idArray;
const tagsDisconnect = idArray;
const tagsDelete = idArray;
const tagsAdd = yup.array().of(tagAddSchema.required()).optional();
const tagsUpdate = yup.array().of(tagUpdateSchema.required()).optional();

//==============================================================
/* #endregion Tag */
//==============================================================

//==============================================================
/* #region Node */
//==============================================================

const conditionField = yup.string().max(2048).optional();
const nodeTypeField = yup.string().oneOf(Object.values(NodeType)).optional();
const wasSuccessfulField = yup.boolean().optional();

/**
 * Data to add or update a combine node's from data
 */
export const nodeDataCombineFrom = yup.object().shape({
    id: idField,
    fromId: idField,
    toId: idField,
})

const nodeDataCombineFromField = yup.array().of(nodeDataCombineFrom.required()).optional();

/**
 * Data to add or update a combine node's data
 */
export const nodeDataCombine = yup.object().shape({
    id: idField,
    toId: idField,
    from: nodeDataCombineFromField,
})

/**
 * Data to add or update a specific condition of a decision node item
 */
export const nodeDataDecisionItemCase = yup.object().shape({
    id: idField,
    condition: conditionField,
})

const nodeDataDecisionItemCaseField = yup.array().of(nodeDataDecisionItemCase.required()).optional();

/**
 * Data to add or update a decision node's item data
 */
export const nodeDataDecisionItem = yup.object().shape({
    id: idField,
    description: descriptionField,
    title: titleField,
    toId: idField,
    when: nodeDataDecisionItemCaseField,
})

const nodeDataDecisionItemField = yup.array().of(nodeDataDecisionItem.required()).optional();

/**
 * Data to add or update a decision node's data
 */
export const nodeDataDecision = yup.object().shape({
    id: idField,
    decisions: nodeDataDecisionItemField,
})

/**
 * Data to add or update an end node's data
 */
export const nodeDataEnd = yup.object().shape({
    id: idField,
    wasSuccessful: wasSuccessfulField,
})

/**
 * Data to add or update a loop node's data TODO complete
 */
export const nodeDataLoop = yup.object().shape({
    id: idField,
})

/**
 * Data to add or update a redirect node's data TODO complete
 */
export const nodeDataRedirect = yup.object().shape({
    id: idField,
})

/**
 * Data to add or update a routine list node's data
 */
export const nodeDataRoutineList = yup.object().shape({
    id: idField,
    isOrdered: yup.boolean().optional(),
    isOptional: yup.boolean().optional(),
    routinesConnect: idArray,
    routinesAdd: resourcesAddField,
    routinesUpdate: resourcesUpdateField,
    routinesRemove: idArray,
})

/**
 * Data to add or update a start node's data
 */
export const nodeDataStart = yup.object().shape({
    id: idField,
})

/**
 * Information required when creating a node
 */
export const nodeAddSchema = yup.object().shape({
    id: idField,
    description: descriptionField,
    title: titleField.required(),
    type: nodeTypeField.required(),
    dataCombine: nodeDataCombine.optional(),
    dataDecision: nodeDataDecision.optional(),
    dataEnd: nodeDataEnd.optional(),
    dataLoop: nodeDataLoop.optional(),
    dataRedirect: nodeDataRedirect.optional(),
    dataRoutineList: nodeDataRoutineList.optional(),
    dataStart: nodeDataStart.optional(),
    nextId: idField,
    previousId: idField,
})

/**
 * Information required when updating a node
 */
 export const nodeUpdateSchema = yup.object().shape({
    id: idField,
    description: descriptionField,
    title: titleField,
    type: nodeTypeField,
    dataCombine: nodeDataCombine.optional(),
    dataDecision: nodeDataDecision.optional(),
    dataEnd: nodeDataEnd.optional(),
    dataLoop: nodeDataLoop.optional(),
    dataRedirect: nodeDataRedirect.optional(),
    dataRoutineList: nodeDataRoutineList.optional(),
    dataStart: nodeDataStart.optional(),
    nextId: idField,
    previousId: idField,
})

//==============================================================
/* #endregion Node */
//==============================================================

//==============================================================
/* #region Organization */
//==============================================================

/**
 * Information required when creating an organization. 
 * You are automatically added as an admin
 */
export const organizationAddSchema = yup.object().shape({
    bio: bioField,
    name: nameField.required(),
    // You are automatically added as an admin. IDs you add here will be requested to be added as a member
    members: idArray,
    resourcesConnect,
    resourcesAdd,
    tagsConnect,
    tagsAdd,
})

/**
 * Information required when updating an organization
 */
export const organizationUpdateSchema = yup.object().shape({
    bio: bioField,
    name: nameField,
    members: idArray,
    resourcesConnect,
    resourcesDisconnect,
    resourcesDelete,
    resourcesAdd,
    resourcesUpdate,
    tagsConnect,
    tagsDisconnect,
    tagsDelete,
    tagsAdd,
    tagsUpdate,
})

//==============================================================
/* #endregion Organization */
//==============================================================

//==============================================================
/* #region Project */
//==============================================================

/**
 * Information required when creating a project. 
 */
export const projectAddSchema = yup.object().shape({
    description: descriptionField,
    name: nameField.required(),
    parentId: idField, // If forked, the parent's id
    createdByUserId: idField, // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: idField, // If associating with an organization you are an admin of, the organization's id
    resourcesConnect,
    resourcesAdd,
    tagsConnect,
    tagsAdd,
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization

/**
 * Information required when updating a project
 */
export const projectUpdateSchema = yup.object().shape({
    description: descriptionField,
    name: nameField,
    userId: idField, // Allows you to request transfer ownership of the project
    organizationId: idField, // Allows you to request transfer ownership of the project
    resourcesConnect,
    resourcesDisconnect,
    resourcesDelete,
    resourcesAdd,
    resourcesUpdate,
    tagsConnect,
    tagsDisconnect,
    tagsDelete,
    tagsAdd,
    tagsUpdate,
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

//==============================================================
/* #endregion Project */
//==============================================================

//==============================================================
/* #region Standard */
//==============================================================

const standardDefaultField = yup.string().max(1024).optional();
const standardSchemaField = yup.string().max(8192).required();
const standardTypeField = yup.string().oneOf(Object.values(StandardType)).optional();

/**
 * Information required when creating a standard. 
 */
export const standardAddSchema = yup.object().shape({
    default: standardDefaultField,
    description: descriptionField,
    isFile: yup.boolean().optional(),
    name: nameField.required(),
    schema: standardSchemaField,
    type: standardTypeField,
    createdByUserId: idField, // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: idField, // If associating with an organization you are an admin of, the organization's id
    tagsConnect,
    tagsAdd,
})

/**
 * Information required when updating a routine
 */
export const standardUpdateSchema = yup.object().shape({
    default: standardDefaultField,
    description: descriptionField,
    isFile: yup.boolean().optional(),
    makingAnonymous: yup.boolean().optional(), // If you want the standard to be made anonymous
    name: nameField.required(),
    schema: standardSchemaField,
    type: standardTypeField,
    tagsConnect,
    tagsDisconnect,
    tagsDelete,
    tagsAdd,
    tagsUpdate,
})

//==============================================================
/* #endregion Standard */
//==============================================================

//==============================================================
/* #region Routine */
//==============================================================

const isAutomatableField = yup.boolean().optional();
const isRequiredField = yup.boolean().optional();
const nodesAddField = yup.array().of(nodeAddSchema.required()).optional();
const routineInstructionsField = yup.string().max(8192).optional();

export const routineInputOutputSchema = yup.object().shape({
    description: descriptionField,
    isRequired: isRequiredField,
    name: nameField,
    standardId: idField, // Connect to existing standard
    standard: standardAddSchema.optional(), // Create new standard
}, [['standardId', 'standard']]) // Makes sure you can't connect to a standard and define a new standard

/**
 * Information required when creating a routine. 
 */
export const routineAddSchema = yup.object().shape({
    description: descriptionField,
    instructions: routineInstructionsField,
    isAutomatable: isAutomatableField,
    title: titleField.required(),
    version: versionField,
    parentId: idField, // If forked, the parent's id
    createdByUserId: idField, // If associating with yourself, your own id. Cannot associate with another user
    createdByOrganizationId: idField, // If associating with an organization you are an admin of, the organization's id
    nodes: idField, // Existing nodes being added to the routine
    nodesAdd: nodesAddField, // New nodes being added to the routine
    inputsAdd: routineInputOutputSchema.optional(), // Routine's inputs
    outputsAdd: routineInputOutputSchema.optional(), // Routine's outputs
    resourcesContextualConnect: resourcesConnect,
    resourcesContextualAdd: resourcesAdd,
    resourcesExternalConnect: resourcesConnect,
    resourcesExternalAdd: resourcesAdd,
    tagsConnect,
    tagsAdd,
}, [['createdByUserId', 'createdByOrganizationId']]) // Makes sure you can't associate with both a user and an organization

/**
 * Information required when updating a routine
 */
export const routineUpdateSchema = yup.object().shape({
    description: descriptionField,
    instructions: routineInstructionsField,
    isAutomatable: isAutomatableField,
    title: titleField.required(),
    version: versionField,
    parentId: idField, // If forked, the parent's id
    userId: idField, // If associating with yourself, your own id. Cannot associate with another user
    organizationId: idField, // If associating with an organization you are an admin of, the organization's id
    nodes: idField, // Existing nodes being added to the routine
    nodesAdd: nodesAddField, // New nodes being added to the routine
    nodesRemove: idArray, // Existing nodes being removed from the routine
    inputsAdd: routineInputOutputSchema.optional(), // Routine's inputs, either update or add
    inputsRemove: idArray, // Routine's inputs to be removed
    outputsAdd: routineInputOutputSchema.optional(), // Routine's outputs, either update or add
    outputsRemove: idArray, // Routine's outputs to be removed
    resourcesContextualConnect: resourcesConnect,
    resourcesContextualDisconnect: resourcesDisconnect,
    resourcesContextualDelete: resourcesDelete,
    resourcesContextualAdd: resourcesAdd,
    resourcesContextualUpdate: resourcesUpdate,
    resourcesExternalConnect: resourcesConnect,
    resourcesExternalDisconnect: resourcesDisconnect,
    resourcesExternalDelete: resourcesDelete,
    resourcesExternalAdd: resourcesAdd,
    resourcesExternalUpdate: resourcesUpdate,
    tagsConnect,
    tagsDisconnect,
    tagsDelete,
    tagsAdd,
    tagsUpdate,
}, [['userId', 'organizationId']]) // Makes sure you can't transfer to both a user and an organization

//==============================================================
/* #endregion Routine */
//==============================================================