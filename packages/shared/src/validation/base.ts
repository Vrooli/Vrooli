/**
 * Shared fields are defined to reduce bugs that may occur when 
 * there is a mismatch between the database and schemas. Every database 
 * field with a duplicate name has the name format, so as long as 
 * that format matches the fields below, there should be no errors.
 */
import * as yup from 'yup';
 
export const id = yup.string().max(256).optional();
export const bio = yup.string().max(2048).optional();
export const description = yup.string().max(2048).optional();
export const language = yup.string().min(2).max(2).optional(); // Language code
export const name = yup.string().min(3).max(128).optional();
export const title = yup.string().min(2).max(128).optional();
export const version = yup.string().max(16).optional();
export const idArray = yup.array().of(id.required()).optional();