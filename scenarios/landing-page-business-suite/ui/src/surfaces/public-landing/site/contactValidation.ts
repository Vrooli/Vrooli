import { siteCopy } from './siteCopy';

const copy = siteCopy.contactPage;
export type ContactField = 'email' | 'subject' | 'message';
export type ContactFieldErrors = Partial<Record<ContactField, string>>;

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;
export const SUBMIT_TIMEOUT_MS = 30000;

export function validateContact(values: { email: string; subject: string; message: string }): ContactFieldErrors {
  const errors: ContactFieldErrors = {};
  const email = values.email.trim();
  if (!email) errors.email = copy.errors.emailRequired;
  else if (!EMAIL.test(email)) errors.email = copy.errors.emailInvalid;
  const subject = values.subject.trim();
  if (!subject) errors.subject = copy.errors.subjectRequired;
  else if (subject.length > 200) errors.subject = copy.errors.subjectLong;
  const message = values.message.trim();
  if (!message) errors.message = copy.errors.messageRequired;
  else if (message.length < 10) errors.message = copy.errors.messageShort;
  return errors;
}
