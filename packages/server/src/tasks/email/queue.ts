import { APP_URL, BUSINESS_NAME } from "@local/shared";
import { PaymentType } from "@prisma/client";
import Bull from "bull";
import fs from "fs";
import { logger } from "../../events/logger.js";
import { HOST, PORT } from "../../redisConn.js";
import { emailProcess } from "./process.js";

let welcomeTemplate = "";
const welcomeTemplateFile = `${process.env.PROJECT_DIR}/packages/server/dist/notify/email/templates/welcome.html`;
if (fs.existsSync(welcomeTemplateFile)) {
    welcomeTemplate = fs.readFileSync(welcomeTemplateFile).toString();
} else {
    logger.error(`Could not find welcome email template at ${welcomeTemplateFile}`);
}

const emailQueue = new Bull("email", { redis: { port: PORT, host: HOST } });
emailQueue.process(emailProcess);

/** Adds an email to a task queue */
export function sendMail(to: string[] = [], subject = "", text = "", html = "", delay = 0) {
    emailQueue.add({
        to,
        subject,
        text,
        html: html.length > 0 ? html : undefined,
    }, { delay });
}

/** Adds a password reset link email to a task queue */
export function sendResetPasswordLink(email: string, userId: string | number, code: string) {
    const link = `${APP_URL}/password-reset?code=${userId}:${code}`;
    emailQueue.add({
        to: [email],
        subject: `${BUSINESS_NAME} Password Reset`,
        text: `A password reset was requested for your account with ${BUSINESS_NAME}. If you sent this request, you may change your password through this link (${link}) to continue. If you did not send this request, please ignore this email.`,
        html: `<p>A password reset was requested for your account with ${BUSINESS_NAME}.</p><p>If you sent this request, you may change your password through this link (<a href="${link}">${link}</a>) to continue.<p>If you did not send this request, please ignore this email.<p>`,
    });
}

/** Adds a verification link email to a task queue */
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

/** Adds a feedback notification email for the admin to a task queue */
export function feedbackNotifyAdmin(text: string, from?: string) {
    emailQueue.add({
        to: [process.env.SITE_EMAIL_USERNAME],
        subject: "Received Vrooli Feedback!",
        text: `Feedback from ${from ?? "anonymous"}: ${text}`,
    });
}

/** Adds a thank you email for a completed payment (not recurring) to a task queue */
export function sendPaymentThankYou(emailAddress: string, paymentType: PaymentType) {
    emailQueue.add({
        to: [emailAddress],
        subject: `Thank you for your ${paymentType === PaymentType.Donation ? "donation" : "purchase"}!`,
        text: paymentType === PaymentType.Donation ?
            `Thank you for your donation to ${BUSINESS_NAME}! Your support is greatly appreciated.` :
            `Thank you for purchasing a premium subscription to ${BUSINESS_NAME}! Your benefits will be available immediately. Thank you for your support!`,
    });
}

/** Adds a payment failed email to a task queue */
export function sendPaymentFailed(emailAddress: string, paymentType: PaymentType) {
    emailQueue.add({
        to: [emailAddress],
        subject: `Your ${paymentType === PaymentType.Donation ? "donation" : "purchase"} failed`,
        text: paymentType === PaymentType.Donation ?
            `Your donation to ${BUSINESS_NAME} failed. Please try again.` :
            `Your purchase of a premium subscription to ${BUSINESS_NAME} failed. Please try again.`,
    });
}

/** Adds a credit card expiring soon warning to a task queue */
export function sendCreditCardExpiringSoon(emailAddress: string) {
    emailQueue.add({
        to: [emailAddress],
        subject: "Your credit card is expiring soon!",
        text: "Your credit card is expiring soon. Please update your payment information to avoid any interruptions to your premium subscription.",
    });
}

/** Adds a "sorry to see you go" email for users who canceled their subscription to a task queue */
export function sendSubscriptionCanceled(emailAddress: string) {
    emailQueue.add({
        to: [emailAddress],
        subject: "Sorry to see you go!",
        text: "We're sorry to see you canceled your subscription. Come back any time!",
    });
}

/** Adds subscription ended email to a task queue */
export function sendSubscriptionEnded(emailAddress: string) {
    emailQueue.add({
        to: [emailAddress],
        subject: "Your subscription has ended",
        text: `Your subscription has ended. If this wasn't intentional, please renew your subscription to continue enjoying premium benefits. Thank you for using ${BUSINESS_NAME}!`,
    });
}
