import { id, maxStringErrorMessage, minStringErrorMessage, requiredErrorMessage } from './base';
import * as yup from 'yup';

const text = yup.string().min(1, minStringErrorMessage).max(8192, maxStringErrorMessage)

/**
 * Information required when submitting feedback
 */
 export const feedbackCreate = yup.object().shape({
    text: text.required(requiredErrorMessage),
    userId: id.notRequired().default(undefined),
})