import { APP_URL, BUSINESS_NAME } from '@shared/consts';
import Bull from 'bull';
import fs from 'fs';
import { HOST, PORT } from '../../redisConn.js';
import { emailProcess } from './process.js';

const welcomeTemplate = fs.readFileSync(`${process.env.PROJECT_DIR}/packages/server/src/notify/email/templates/welcome.html`).toString();

const emailQueue = new Bull('email', { redis: { port: PORT, host: HOST } });
emailQueue.process(emailProcess);

export function sendMail(to: string[] = [], subject = '', text = '', html = '', delay: number = 0) {
    emailQueue.add({
        to: to,
        subject: subject,
        text: text,
        html: html.length > 0 ? html : undefined,
    }, { delay });
}

export function sendResetPasswordLink(email: string, userId: string | number, code: string) {
    const link = `${APP_URL}/password-reset?code=${userId}:${code}`;
    emailQueue.add({
        to: [email],
        subject: `${BUSINESS_NAME} Password Reset`,
        text: `A password reset was requested for your account with ${BUSINESS_NAME}. If you sent this request, you may change your password through this link (${link}) to continue. If you did not send this request, please ignore this email.`,
        html: `<p>A password reset was requested for your account with ${BUSINESS_NAME}.</p><p>If you sent this request, you may change your password through this link (<a href=\"${link}\">${link}</a>) to continue.<p>If you did not send this request, please ignore this email.<p>`
    });
}

export function sendVerificationLink(email: string, userId: string | number, code: string) {
    // Replace all "${VERIFY_LINK}" in welcomeTemplate with the the actual link
    const link = `${APP_URL}/start?code=${userId}:${code}`;
    const html = welcomeTemplate.replace(/\$\{VERIFY_LINK\}/g, link);
    emailQueue.add({
        to: [email],
        subject: `Verify ${BUSINESS_NAME} Account`,
        text: `Welcome to ${BUSINESS_NAME}! Please log in through this link (${link}) to verify your account. If you did not create an account with us, please ignore this link.`,
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