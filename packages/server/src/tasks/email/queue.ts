import { BUSINESS_NAME, LINKS, PaymentType } from "@local/shared";
import Bull from "bull";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";
import { CustomError } from "../../events/error";

export type EmailProcessPayload = {
    to: string[];
    subject: string;
    text: string;
    html?: string;
}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let UI_URL: string;
let emailProcess: (job: Bull.Job<EmailProcessPayload>) => Promise<unknown>;
let emailQueue: Bull.Queue<EmailProcessPayload>;
let welcomeTemplate: string;
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export const setupEmailQueue = async () => {
    try {
        const loggerPath = path.join(dirname, "../../events/logger" + importExtension);
        const loggerModule = await import(loggerPath);
        logger = loggerModule.logger;

        const redisConnPath = path.join(dirname, "../../redisConn" + importExtension);
        const redisConnModule = await import(redisConnPath);
        HOST = redisConnModule.HOST;
        PORT = redisConnModule.PORT;

        const serverPath = path.join(dirname, "../../server" + importExtension);
        const serverModule = await import(serverPath);
        UI_URL = serverModule.UI_URL;

        const processPath = path.join(dirname, "./process" + importExtension);
        const processModule = await import(processPath);
        emailProcess = processModule.emailProcess;

        // Initialize the Bull queue
        emailQueue = new Bull<EmailProcessPayload>("email", {
            redis: { port: PORT, host: HOST },
        });
        emailQueue.process(emailProcess);

        // Load templates
        const emailTemplatePath = path.join(dirname, "../../../dist/tasks/email/templates/welcome.html");
        if (fs.existsSync(emailTemplatePath)) {
            welcomeTemplate = fs.readFileSync(emailTemplatePath).toString();
        } else {
            logger.error(`Could not find welcome email template at ${emailTemplatePath}`);
            welcomeTemplate = "";
        }
    } catch (error) {
        const errorMessage = "Failed to setup email queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0204", error });
        } else {
            console.error(errorMessage, error);
        }
    }
};

/** Adds an email to a task queue */
export const sendMail = (
    to: string[],
    subject: string,
    text: string,
    html = "",
    delay = 0,
) => {
    // Must include at least one "to" email address
    if (to.length === 0) {
        throw new CustomError("0354", "InternalError", ["en"]);
    }
    emailQueue.add({
        to,
        subject,
        text,
        html: html.length > 0 ? html : undefined,
    }, { delay });
};

/** Adds a password reset link email to a task queue */
export const sendResetPasswordLink = (email: string, userId: string, code: string) => {
    const link = `${UI_URL}${LINKS.ResetPassword}?code=${userId}:${code}`;
    emailQueue.add({
        to: [email],
        subject: `${BUSINESS_NAME} Password Reset`,
        text: `A password reset was requested for your account with ${BUSINESS_NAME}. If you sent this request, you may change your password through this link (${link}) to continue. If you did not send this request, please ignore this email.`,
        html: `<p>A password reset was requested for your account with ${BUSINESS_NAME}.</p><p>If you sent this request, you may change your password through this link (<a href="${link}">${link}</a>) to continue.<p>If you did not send this request, please ignore this email.<p>`,
    });
};

/** Adds a verification link email to a task queue */
export const sendVerificationLink = (email: string, userId: string, code: string) => {
    // Replace all "${VERIFY_LINK}" in welcomeTemplate with the the actual link
    const link = `${UI_URL}${LINKS.Login}?code=${userId}:${code}`;
    const html = welcomeTemplate.replace(/\$\{VERIFY_LINK\}/g, link);
    emailQueue.add({
        to: [email],
        subject: `Verify ${BUSINESS_NAME} Account`,
        text: `Welcome to ${BUSINESS_NAME}! Please log in through [this link](${link}) to verify your account. If you did not create an account with us, please ignore this link.`,
        html: html.length > 0 ? html : undefined,
    });
};

/** Adds a feedback notification email for the admin to a task queue */
export const feedbackNotifyAdmin = (text: string, from?: string) => {
    emailQueue.add({
        to: [process.env.SITE_EMAIL_USERNAME ?? ""],
        subject: "Received Vrooli Feedback!",
        text: `Feedback from ${from ?? "anonymous"}: ${text}`,
    });
};

/** Adds a thank you email for a completed payment (not recurring) to a task queue */
export const sendPaymentThankYou = (emailAddress: string, paymentType: PaymentType) => {
    emailQueue.add({
        to: [emailAddress],
        subject: `Thank you for your ${paymentType === PaymentType.Donation ? "donation" : "purchase"}!`,
        text: paymentType === PaymentType.Donation ?
            `Thank you for your donation to ${BUSINESS_NAME}! Your support is greatly appreciated.` :
            `Thank you for purchasing a premium subscription to ${BUSINESS_NAME}! Your benefits will be available immediately. Thank you for your support!`,
    });
};

/** Adds a payment failed email to a task queue */
export const sendPaymentFailed = (emailAddress: string, paymentType: PaymentType) => {
    emailQueue.add({
        to: [emailAddress],
        subject: `Your ${paymentType === PaymentType.Donation ? "donation" : "purchase"} failed`,
        text: paymentType === PaymentType.Donation ?
            `Your donation to ${BUSINESS_NAME} failed. Please try again.` :
            `Your purchase of a premium subscription to ${BUSINESS_NAME} failed. Please try again.`,
    });
};

/** Adds a credit card expiring soon warning to a task queue */
export const sendCreditCardExpiringSoon = (emailAddress: string) => {
    emailQueue.add({
        to: [emailAddress],
        subject: "Your credit card is expiring soon!",
        text: "Your credit card is expiring soon. Please update your payment information to avoid any interruptions to your premium subscription.",
    });
};

/** Adds a "sorry to see you go" email for users who canceled their subscription to a task queue */
export const sendSubscriptionCanceled = (emailAddress: string) => {
    emailQueue.add({
        to: [emailAddress],
        subject: "Sorry to see you go!",
        text: "We're sorry to see you canceled your subscription. Come back any time!",
    });
};

/** Adds subscription ended email to a task queue */
export const sendSubscriptionEnded = (emailAddress: string) => {
    emailQueue.add({
        to: [emailAddress],
        subject: "Your subscription has ended",
        text: `Your subscription has ended. If this wasn't intentional, please renew your subscription to continue enjoying premium benefits. Thank you for using ${BUSINESS_NAME}!`,
    });
};
