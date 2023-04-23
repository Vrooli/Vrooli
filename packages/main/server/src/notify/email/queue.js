import { APP_URL, BUSINESS_NAME } from "@local/consts";
import Bull from "bull";
import fs from "fs";
import { HOST, PORT } from "../../redisConn.js";
import { emailProcess } from "./process.js";
const welcomeTemplate = fs.readFileSync(`${process.env.PROJECT_DIR}/packages/server/dist/notify/email/templates/welcome.html`).toString();
const emailQueue = new Bull("email", { redis: { port: PORT, host: HOST } });
emailQueue.process(emailProcess);
export function sendMail(to = [], subject = "", text = "", html = "", delay = 0) {
    emailQueue.add({
        to,
        subject,
        text,
        html: html.length > 0 ? html : undefined,
    }, { delay });
}
export function sendResetPasswordLink(email, userId, code) {
    const link = `${APP_URL}/password-reset?code=${userId}:${code}`;
    emailQueue.add({
        to: [email],
        subject: `${BUSINESS_NAME} Password Reset`,
        text: `A password reset was requested for your account with ${BUSINESS_NAME}. If you sent this request, you may change your password through this link (${link}) to continue. If you did not send this request, please ignore this email.`,
        html: `<p>A password reset was requested for your account with ${BUSINESS_NAME}.</p><p>If you sent this request, you may change your password through this link (<a href=\"${link}\">${link}</a>) to continue.<p>If you did not send this request, please ignore this email.<p>`,
    });
}
export function sendVerificationLink(email, userId, code) {
    const link = `${APP_URL}/start?code=${userId}:${code}`;
    const html = welcomeTemplate.replace(/\$\{VERIFY_LINK\}/g, link);
    emailQueue.add({
        to: [email],
        subject: `Verify ${BUSINESS_NAME} Account`,
        text: `Welcome to ${BUSINESS_NAME}! Please log in through this link (${link}) to verify your account. If you did not create an account with us, please ignore this link.`,
        html,
    });
}
export function feedbackNotifyAdmin(text, from) {
    emailQueue.add({
        to: [process.env.SITE_EMAIL_USERNAME],
        subject: "Received Vrooli Feedback!",
        text: `Feedback from ${from ?? "anonymous"}: ${text}`,
    });
}
//# sourceMappingURL=queue.js.map