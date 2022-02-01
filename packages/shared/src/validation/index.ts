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
 
 //==============================================================
 /* #region Shared fields */
 //==============================================================
 // Shared fields are defined to reduce bugs that may occur when 
 // there is a mismatch between the database and schemas. Every database 
 // field with a duplicate name has the name format, so as long as 
 // that format matches the fields below, there should be no errors.
 //==============================================================
 
 export const id = yup.string().max(256).optional();
 export const bio = yup.string().max(2048).optional();
 export const description = yup.string().max(2048).optional();
 export const name = yup.string().max(128).optional();
 export const title = yup.string().max(128).optional();
 export const version = yup.string().max(16).optional();
 export const idArray = yup.array().of(id.required()).optional();

//==============================================================
/* #endregion Shared Fields */
//==============================================================

//==============================================================
/* #region Exports */
//==============================================================

export * from './node';
export * from './organization';
export * from './project';
export * from './resource';
export * from './routine';
export * from './standard';
export * from './tag';
export * from './user';
 
//==============================================================
/* #endregion Exports */
//==============================================================