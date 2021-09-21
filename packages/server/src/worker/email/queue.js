import Bull from 'bull';
import { emailProcess } from './process';
import fs from 'fs';
const { BUSINESS_NAME, WEBSITE } = JSON.parse(fs.readFileSync(`${process.env.PROJECT_DIR}/assets/public/business.json`, 'utf8'));

const emailQueue = new Bull('email', { redis: { 
    port: process.env.REDIS_CONN.split(':')[1],
    host: process.env.REDIS_CONN.split(':')[0]
} });
emailQueue.process(emailProcess);

export function sendMail(to=[], subject='', text='', html='') {
    emailQueue.add({
        to: to,
        subject: subject,
        text: text,
        html: html
    });
}

export function customerNotifyAdmin(name) {
    emailQueue.add({
        to: [process.env.SITE_EMAIL_USERNAME],
        subject: `Account created for ${name}`,
        text: `${name} has created an account with ${BUSINESS_NAME.Long}. Website accounts can be viewed at ${WEBSITE}/admin/customers`,
        html: `<p>${name} has created an account with ${BUSINESS_NAME.Long}. Website accounts can be viewed at <a href=\"${WEBSITE}/admin/customers\">${WEBSITE}/admin/customers</a></p>`
    });
}

export function sendResetPasswordLink(email, customer_id, code) {
    emailQueue.add({
        to: [email],
        subject: `${BUSINESS_NAME.Short} Password Reset`,
        text: `A password reset was requested for your account with ${BUSINESS_NAME.Long}. If you sent this request, you may change your password through this link (${WEBSITE}/password-reset/${customer_id}/${code}) to continue. If you did not send this request, please ignore this email.`,
        html: `<p>A password reset was requested for your account with ${BUSINESS_NAME.Long}.</p><p>If you sent this request, you may change your password through this link (<a href=\"${WEBSITE}/password-reset/${customer_id}/${code}\">${WEBSITE}/password-reset/${customer_id}/${code}</a>) to continue.<p>If you did not send this request, please ignore this email.<p>`
    });
}

export function confirmJoinWaitlist(email, confirmationCode) {
    emailQueue.add({
        to: [email],
        subject: `Confirm your spot on the ${BUSINESS_NAME.Short} waitlist!`,
        text: `Please click this link (${WEBSITE}/join-us/${confirmationCode}) to confirm your spot on the ${BUSINESS_NAME.Short} waitlist.`,
        html: `<p>Please click this link (<a href=\"${WEBSITE}/join-us/${confirmationCode}\">${WEBSITE}/join-us/${confirmationCode}</a>) to confirm your spot on the waitlist.</p>`
    });
}

export function joinedWaitlist(email) {
    emailQueue.add({
        to: [email],
        subject: `You're on the waitlist for ${BUSINESS_NAME.Short}!`,
        text: `Congratulations! You're on the waitlist for Vrooli. We'll let you know when the site is ready :)`,
        html: `<p>Congratulations!</p><p>You're on the waitlist for Vrooli.</p><p>We'll let you know when the site is ready :)<p>`
    });
}

export function joinWaitlistNotifyAdmin(name) {
    emailQueue.add({
        to: [process.env.SITE_EMAIL_USERNAME],
        subject: `${name} joined the ${BUSINESS_NAME.Short} waitlist!`,
        text: `${name} has joined the ${BUSINESS_NAME.Short} waitlist! It's catching steam :)`,
        html: `<p>${name} has joined the ${BUSINESS_NAME.Short} waitlist!</p><p>It's catching steam :)<p>`
    });
}

export function sendVerificationLink(email, customer_id) {
    emailQueue.add({
        to: [email],
        subject: `Verify ${BUSINESS_NAME.Short} Account`,
        text: `Welcome to ${BUSINESS_NAME.Long}! Please login through this link (${WEBSITE}/login/${customer_id}) to verify your account. If you did not create an account with us, please ignore this link.`,
        html: `<p>Welcome to ${BUSINESS_NAME.Long}!</p><p>Please login through this link (<a href=\"${WEBSITE}/login/${customer_id}\">${WEBSITE}/login/${customer_id}</a>) to verify your account.</p><p>If you did not create an account with us, please ignore this message.</p>`
    });
}