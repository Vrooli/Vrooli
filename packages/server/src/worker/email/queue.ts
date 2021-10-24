import Bull from 'bull';
import { emailProcess } from './process.js';
import fs from 'fs';
import { HOST, PORT } from '../connection.js';
const { BUSINESS_NAME, WEBSITE } = JSON.parse(fs.readFileSync(`${process.env.PROJECT_DIR}/assets/public/business.json`, 'utf8'));

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
        text: `${username} has created an account with ${BUSINESS_NAME.Long}. Website accounts can be viewed at ${WEBSITE}/admin/customers`,
        html: `<p>${username} has created an account with ${BUSINESS_NAME.Long}. Website accounts can be viewed at <a href=\"${WEBSITE}/admin/customers\">${WEBSITE}/admin/customers</a></p>`
    });
}

export function sendResetPasswordLink(email: string, customer_id: string | number, code: string) {
    emailQueue.add({
        to: [email],
        subject: `${BUSINESS_NAME.Short} Password Reset`,
        text: `A password reset was requested for your account with ${BUSINESS_NAME.Long}. If you sent this request, you may change your password through this link (${WEBSITE}/password-reset/${customer_id}/${code}) to continue. If you did not send this request, please ignore this email.`,
        html: `<p>A password reset was requested for your account with ${BUSINESS_NAME.Long}.</p><p>If you sent this request, you may change your password through this link (<a href=\"${WEBSITE}/password-reset/${customer_id}/${code}\">${WEBSITE}/password-reset/${customer_id}/${code}</a>) to continue.<p>If you did not send this request, please ignore this email.<p>`
    });
}

export function sendVerificationLink(email: string, customer_id: string | number) {
    emailQueue.add({
        to: [email],
        subject: `Verify ${BUSINESS_NAME.Short} Account`,
        text: `Welcome to ${BUSINESS_NAME.Long}! Please login through this link (${WEBSITE}/login/${customer_id}) to verify your account. If you did not create an account with us, please ignore this link.`,
        html: `<p>Welcome to ${BUSINESS_NAME.Long}!</p><p>Please login through this link (<a href=\"${WEBSITE}/login/${customer_id}\">${WEBSITE}/login/${customer_id}</a>) to verify your account.</p><p>If you did not create an account with us, please ignore this message.</p>`
    });
}

export function confirmJoinWaitlist(email: string, confirmationCode: string) {
    emailQueue.add({
        to: [email],
        subject: `Confirm your spot on the ${BUSINESS_NAME.Short} waitlist!`,
        text: `Please click this link (${WEBSITE}/join-us/${confirmationCode}) to confirm your spot on the ${BUSINESS_NAME.Short} waitlist.`,
        html: `<p>Please click this link (<a href=\"${WEBSITE}/join-us/${confirmationCode}\">${WEBSITE}/join-us/${confirmationCode}</a>) to confirm your spot on the waitlist.</p>`
    });
}

export function joinedWaitlist(email: string) {
    emailQueue.add({
        to: [email],
        subject: `You're on the waitlist for ${BUSINESS_NAME.Short}!`,
        text: `Congratulations! You're on the waitlist for Vrooli. We'll let you know when the site is ready :)`,
        html: `<p>Congratulations!</p><p>You're on the waitlist for Vrooli.</p><p>We'll let you know when the site is ready :)<p>`
    });
}

export function joinWaitlistNotifyAdmin(username: string) {
    emailQueue.add({
        to: [process.env.SITE_EMAIL_USERNAME],
        subject: `${username} joined the ${BUSINESS_NAME.Short} waitlist!`,
        text: `${username} has joined the ${BUSINESS_NAME.Short} waitlist! It's catching steam :)`,
        html: `<p>${username} has joined the ${BUSINESS_NAME.Short} waitlist!</p><p>It's catching steam :)<p>`
    });
}