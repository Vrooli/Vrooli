import { id } from './base';
import * as yup from 'yup';

const text = yup.string().min(1).max(8192)

/**
 * Information required when submitting feedback
 */
 export const feedbackCreate = yup.object().shape({
    text: text.required(),
    userId: id.notRequired().default(undefined),
})