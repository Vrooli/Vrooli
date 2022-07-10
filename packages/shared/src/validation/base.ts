/**
 * Shared fields are defined to reduce bugs that may occur when 
 * there is a mismatch between the database and schemas. Every database 
 * field with a duplicate name has the name format, so as long as 
 * that format matches the fields below, there should be no errors.
 */
import * as yup from 'yup';
 
export const id = yup.string().max(256)
export const bio = yup.string().max(2048)
export const description = yup.string().max(2048)
export const language = yup.string().min(2).max(3) // Language code
export const name = yup.string().min(3).max(128)
export const handle = yup.string().min(3).max(16).nullable() // ADA Handle
export const tag = yup.string().min(2).max(64)
export const title = yup.string().min(2).max(128)
export const version = yup.string().max(16)
export const idArray = yup.array().of(id.required())
export const tagArray = yup.array().of(tag.required())