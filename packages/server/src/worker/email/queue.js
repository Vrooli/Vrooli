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

export function orderNotifyAdmin() {
    emailQueue.add({
        to: [process.env.SITE_EMAIL_USERNAME],
        subject: "New Order Received!",
        text: `A new order has been submitted. It can be viewed at ${WEBSITE}/admin/orders`,
        html: `<p>A new order has been submitted. It can be viewed at <a href=\"${WEBSITE}/admin/orders\">${WEBSITE}/admin/orders</a></p>`
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

export function sendVerificationLink(email, customer_id) {
    emailQueue.add({
        to: [email],
        subject: `Verify ${BUSINESS_NAME.Short} Account`,
        text: `Welcome to ${BUSINESS_NAME.Long}! Please login through this link (${WEBSITE}/login/${customer_id}) to verify your account. If you did not create an account with us, please ignore this link.`,
        html: `<p>Welcome to ${BUSINESS_NAME.Long}!</p><p>Please login through this link (<a href=\"${WEBSITE}/login/${customer_id}\">${WEBSITE}/login/${customer_id}</a>) to verify your account.</p><p>If you did not create an account with us, please ignore this message.</p>`
    });
}