import { id } from './base';
import * as yup from 'yup';

const text = yup.string().min(1).max(8192).optional();

/**
 * Information required when submitting feedback
 */
 export const feedbackAdd = yup.object().shape({
    text: text.required(),
    userId: id,
})