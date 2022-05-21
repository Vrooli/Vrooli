import Bull from 'bull';
import { emailProcess } from './process.js';
import { APP_URL, BUSINESS_NAME } from '@local/shared';
import fs from 'fs';

const welcomeTemplate = fs.readFileSync(`${process.env.PROJECT_DIR}/packages/server/src/worker/email/templates/welcome.html`).toString();

export function sendMail(to=[], subject='', text='', html='') {
}

export function sendResetPasswordLink(email: string, userId: string | number, code: string) {
}

export function sendVerificationLink(email: string, userId: string | number, code: string) {
    // Replace all "${VERIFY_LINK}" in welcomeTemplate with the the actual link
    const link = `${APP_URL}/start?code=${userId}:${code}`;
    const html = welcomeTemplate.replace(/\$\{VERIFY_LINK\}/g, link);
}

export function feedbackNotifyAdmin(text: string, from?: string) {
}