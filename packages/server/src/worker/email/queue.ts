import Bull from 'bull';
import { emailProcess } from './process.js';
import { HOST, PORT } from '../../redisConn';
import { APP_URL, BUSINESS_NAME } from '@local/shared';
import fs from 'fs';

const welcomeTemplate = fs.readFileSync(`${process.env.PROJECT_DIR}/packages/server/src/worker/email/templates/welcome.html`).toString();

const emailQueue = new Bull('email', { redis: { port: PORT, host: HOST } });
emailQueue.process(emailProcess);

export function sendMail(to=[], subject='', text='', html='') {
    emailQueue.add({
        to: to,
        subject: subject,
        text: text,
        html: html
    });
}

export function sendResetPasswordLink(email: string, userId: string | number, code: string) {
    emailQueue.add({
        to: [email],
        subject: `${BUSINESS_NAME} Password Reset`,
        text: `A password reset was requested for your account with ${BUSINESS_NAME}. If you sent this request, you may change your password through this link (${APP_URL}/password-reset/${userId}/${code}) to continue. If you did not send this request, please ignore this email.`,
        html: `<p>A password reset was requested for your account with ${BUSINESS_NAME}.</p><p>If you sent this request, you may change your password through this link (<a href=\"${APP_URL}/password-reset/${userId}/${code}\">${APP_URL}/password-reset/${userId}/${code}</a>) to continue.<p>If you did not send this request, please ignore this email.<p>`
    });
}

export function sendVerificationLink(email: string, userId: string | number, code: string) {
    // Replace all "${VERIFY_LINK}" in welcomeTemplate with the the actual link
    const link = `${APP_URL}/start?code=${userId}:${code}`;
    const html = welcomeTemplate.replace(/\$\{VERIFY_LINK\}/g, link);
    emailQueue.add({
        to: [email],
        subject: `Verify ${BUSINESS_NAME} Account`,
        text: `Welcome to ${BUSINESS_NAME}! Please log in through this link (${APP_URL}/start?code=${userId}:${code}) to verify your account. If you did not create an account with us, please ignore this link.`,
        html,
    });
}

export function feedbackNotifyAdmin(text: string, from?: string) {
    emailQueue.add({
        to: [process.env.SITE_EMAIL_USERNAME],
        subject: `Received Vrooli Feedback!`,
        text: `Feedback from ${from ?? 'anonymous'}: ${text}`,
    });
}