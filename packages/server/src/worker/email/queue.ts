import Bull from 'bull';
import { emailProcess } from './process.js';
import { HOST, PORT } from '../connection.js';
import { BUSINESS_NAME, WEBSITE } from '@local/shared'

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

export function customerNotifyAdmin(username: string) {
    emailQueue.add({
        to: [process.env.SITE_EMAIL_USERNAME],
        subject: `Account created for ${username}`,
        text: `${username} has created an account with ${BUSINESS_NAME}. Website accounts can be viewed at ${WEBSITE}/admin/customers`,
        html: `<p>${username} has created an account with ${BUSINESS_NAME}. Website accounts can be viewed at <a href=\"${WEBSITE}/admin/customers\">${WEBSITE}/admin/customers</a></p>`
    });
}

export function sendResetPasswordLink(email: string, customer_id: string | number, code: string) {
    emailQueue.add({
        to: [email],
        subject: `${BUSINESS_NAME} Password Reset`,
        text: `A password reset was requested for your account with ${BUSINESS_NAME}. If you sent this request, you may change your password through this link (${WEBSITE}/password-reset/${customer_id}/${code}) to continue. If you did not send this request, please ignore this email.`,
        html: `<p>A password reset was requested for your account with ${BUSINESS_NAME}.</p><p>If you sent this request, you may change your password through this link (<a href=\"${WEBSITE}/password-reset/${customer_id}/${code}\">${WEBSITE}/password-reset/${customer_id}/${code}</a>) to continue.<p>If you did not send this request, please ignore this email.<p>`
    });
}

export function sendVerificationLink(email: string, customer_id: string | number) {
    emailQueue.add({
        to: [email],
        subject: `Verify ${BUSINESS_NAME} Account`,
        text: `Welcome to ${BUSINESS_NAME}! Please login through this link (${WEBSITE}/login/${customer_id}) to verify your account. If you did not create an account with us, please ignore this link.`,
        html: `<p>Welcome to ${BUSINESS_NAME}!</p><p>Please login through this link (<a href=\"${WEBSITE}/login/${customer_id}\">${WEBSITE}/login/${customer_id}</a>) to verify your account.</p><p>If you did not create an account with us, please ignore this message.</p>`
    });
}

export function feedbackNotifyAdmin(text: string, from?: string) {
    emailQueue.add({
        to: [process.env.SITE_EMAIL_USERNAME],
        subject: `Received Vrooli Feedback!`,
        text: `Feedback from ${from ?? 'anonymous'}: ${text}`,
    });
}