import { BUSINESS_NAME, LINKS } from "@local/shared";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { logger } from "../../events/logger.js";
import { UI_URL } from "../../server.js";
import { QueueTaskType } from "../taskTypes.js";

const dirname = path.dirname(fileURLToPath(import.meta.url));

let welcomeTemplate: string;

// Load templates
const emailTemplatePath = path.join(dirname, "../../../dist/tasks/email/templates/welcome.html");
if (fs.existsSync(emailTemplatePath)) {
    welcomeTemplate = fs.readFileSync(emailTemplatePath).toString();
} else {
    logger.error(`Could not find welcome email template at ${emailTemplatePath}`);
    welcomeTemplate = "";
}

export const AUTH_EMAIL_TEMPLATES = {
    ResetPassword: (userId: string, link: string) => ({
        type: QueueTaskType.EMAIL_SEND as const,
        id: `${userId}:reset-password`, // Override existing job if it exists
        subject: `${BUSINESS_NAME} Password Reset`,
        text: `A password reset was requested for your account with ${BUSINESS_NAME}. If you sent this request, you may change your password through this link (${link}) to continue. If you did not send this request, please ignore this email.`,
        html: `<p>A password reset was requested for your account with ${BUSINESS_NAME}.</p><p>If you sent this request, you may change your password through this link (<a href="${link}">${link}</a>) to continue.<p>If you did not send this request, please ignore this email.<p>`,
    }),
    VerificationLink: (userId: string, link: string) => {
        // Replace all "${VERIFY_LINK}" in welcomeTemplate with the the actual link
        const html = welcomeTemplate.replace(/\$\{VERIFY_LINK\}/g, link);
        return {
            type: QueueTaskType.EMAIL_SEND as const,
            id: `${userId}:send-verification-link`, // Override existing job if it exists
            subject: `Verify ${BUSINESS_NAME} Account`,
            text: `Welcome to ${BUSINESS_NAME}! Please log in through [this link](${link}) to verify your account. If you did not create an account with us, please ignore this link.`,
            html: html.length > 0 ? html : undefined,
        };
    },
    FeedbackNotifyAdmin: (userId: string, text: string) => ({
        type: QueueTaskType.EMAIL_SEND as const,
        id: `${userId}-feedback-notify-admin`, // Override existing job if it exists
        subject: "Received Vrooli Feedback!",
        text: `Feedback received: ${text}`,
    }),
    PaymentThankYou: (userId: string, isDonation: boolean) => ({
        type: QueueTaskType.EMAIL_SEND as const,
        id: `${userId}-payment-thank-you-donation`, // Override existing job if it exists
        subject: `Thank you for your ${isDonation ? "donation" : "purchase"}!`,
        text: isDonation ?
            `Thank you for your donation to ${BUSINESS_NAME}! Your support is greatly appreciated.` :
            `Thank you for purchasing a premium subscription to ${BUSINESS_NAME}! Your benefits will be available immediately. Thank you for your support!`,
    }),
    PaymentFailed: (userId: string, isDonation: boolean) => ({
        type: QueueTaskType.EMAIL_SEND as const,
        id: `${userId}-payment-failed`, // Override existing job if it exists
        subject: `Your ${isDonation ? "donation" : "purchase"} failed`,
        text: isDonation ?
            `Your donation to ${BUSINESS_NAME} failed. Please try again.` :
            `Your purchase of a premium subscription to ${BUSINESS_NAME} failed. Please try again.`,
    }),
    CreditCardExpiringSoon: (userId: string) => ({
        type: QueueTaskType.EMAIL_SEND as const,
        id: `${userId}-credit-card-expiring-soon`, // Override existing job if it exists
        subject: "Your credit card is expiring soon!",
        text: "Your credit card is expiring soon. Please update your payment information to avoid any interruptions to your premium subscription.",
    }),
    SubscriptionCanceled: (userId: string) => ({
        type: QueueTaskType.EMAIL_SEND as const,
        id: `${userId}-subscription-canceled`, // Override existing job if it exists
        subject: "Sorry to see you go!",
        text: "We're sorry to see you canceled your subscription. Come back any time!",
    }),
    SubscriptionEnded: (userId: string) => ({
        type: QueueTaskType.EMAIL_SEND as const,
        id: `${userId}-subscription-ended`, // Override existing job if it exists
        subject: "Your subscription has ended",
        text: `Your subscription has ended. If this wasn't intentional, please renew your subscription to continue enjoying premium benefits. Thank you for using ${BUSINESS_NAME}!`,
    }),
    TrialEndingSoon: (userId: string) => {
        const link = `${UI_URL}${LINKS.Pro}`;
        return {
            type: QueueTaskType.EMAIL_SEND as const,
            id: `${userId}:send-trial-ending-soon`, // Override existing job if it exists
            subject: "Your trial is ending soon!",
            text: `Your free trial is ending soon. Upgrade to a pro subscription to continue receiving benefits, such as free monthly credits and increased limits. ${link}`,
        };
    },
};
