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

export * from './base';
export * from './comment';
export * from './email';
export * from './feedback';
export * from './inputs';
export * from './node';
export * from './organization';
export * from './project';
export * from './report';
export * from './resource';
export * from './resourceList';
export * from './role';
export * from './routine';
export * from './runInputs';
export * from './standard';
export * from './step';
export * from './run';
export * from './tag';
export * from './tagHidden';
export * from './user';
export * from './wallet';