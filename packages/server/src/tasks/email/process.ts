import { type Job } from "bullmq";
import * as nodemailer from "nodemailer";
import { logger } from "../../events/logger.js";
import { type EmailTask } from "../taskTypes.js";

const HOST = "smtp.gmail.com";
const PORT = 465;

let transporter: nodemailer.Transporter | null = null;
// Flag to track if setup previously failed, for any critical reason.
// This helps prevent repeated initialization attempts and log spam.
let criticalSetupFailed = false;

/**
 * Function to setup transporter. This is needed because 
 * the password is loaded from the secrets location, so it's 
 * not available at startup.
 */
export function setupTransporter() {
    const emailUsername = process.env.SITE_EMAIL_USERNAME;
    const emailPassword = process.env.SITE_EMAIL_PASSWORD;

    // Step 1: Check credentials and manage criticalSetupFailed state
    if (!emailUsername || !emailPassword) {
        if (!criticalSetupFailed) { // Log detailed error only on first detection or transition to critical
            logger.error(
                "Email transporter setup failed: SITE_EMAIL_USERNAME or SITE_EMAIL_PASSWORD is not defined in environment variables. Transporter cannot be initialized.",
                { trace: "email.transporter.initFailedCredentialsMissing" },
            );
        } else {
            // Credentials still missing, and we already knew there was a critical failure.
            // Skip repeated warnings - already logged on first detection
        }
        criticalSetupFailed = true;
        transporter = null; // Ensure transporter is cleared if credentials are (now) missing
        return;
    }

    // Credentials ARE present.
    // If criticalSetupFailed was true, it means credentials were missing but are now available.
    if (criticalSetupFailed) {
        criticalSetupFailed = false; // Reset flag, as the critical issue (missing creds) is resolved.
        // Force re-initialization by setting transporter to null, ensuring we use the now-available credentials.
        transporter = null;
    }

    // Step 2: If already initialized (and credentials are fine, criticalSetupFailed is false), nothing more to do.
    if (transporter !== null) {
        return;
    }

    // Step 3: Attempt to create transporter.
    // At this point:
    // - emailUsername and emailPassword ARE defined.
    // - criticalSetupFailed is false.
    // - transporter is null (either never initialized, or reset above because credentials reappeared).
    try {
        transporter = nodemailer.createTransport({
            host: HOST,
            port: PORT,
            secure: true,
            auth: {
                user: emailUsername, // Use validated variables
                pass: emailPassword, // Use validated variables
            },
        });
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        logger.error(
            "Email transporter setup failed during nodemailer.createTransport. Transporter remains uninitialized. This may be due to network issues or incorrect (but present) credentials.",
            {
                trace: "email.transporter.createTransportFailed",
                error: errorMessage, // Log the specific error from createTransport
            },
        );
        transporter = null; // Ensure transporter is null if createTransport fails.
        // criticalSetupFailed remains false because this is not a missing-credential issue.
    }
}

/**
 * Process an email job from the queue
 * 
 * @param job The email job to process
 * @returns Result with success indicator and email info
 */
export async function emailProcess(job: Job<EmailTask>) {
    try {
        setupTransporter();

        if (!transporter) {
            logger.error(
                "Email transporter is null after setup attempt. This might be due to missing SITE_EMAIL_USERNAME, SITE_EMAIL_PASSWORD, or an unexpected issue during nodemailer.createTransport if it did not throw an error.",
                { jobId: job.id, trace: "email.transporter.initFailedNull" },
            );
            return { success: false };
        }

        // Assign the now-guaranteed non-null transporter to a const
        // to satisfy stricter linting and improve type clarity for this scope.
        const activeTransporter = transporter;

        const siteEmailFromName = process.env.SITE_EMAIL_FROM;
        const emailUser = process.env.SITE_EMAIL_ALIAS || process.env.SITE_EMAIL_USERNAME;

        if (!siteEmailFromName) {
            logger.error(
                "Email configuration error: SITE_EMAIL_FROM is not defined. Cannot send email.",
                { jobId: job.id, trace: "email.config.missingFromName" },
            );
            return { success: false };
        }

        if (!emailUser) {
            logger.error(
                "Email configuration error: Neither SITE_EMAIL_ALIAS nor SITE_EMAIL_USERNAME is defined. Cannot send email.",
                { jobId: job.id, trace: "email.config.missingEmailUser" },
            );
            return { success: false };
        }

        const fromAddress = `"${siteEmailFromName}" <${emailUser}>`;
        const info = await activeTransporter.sendMail({
            from: fromAddress,
            to: job.data.to.join(", "),
            subject: job.data.subject,
            text: job.data.text,
            html: job.data.html,
        });

        return {
            success: info.rejected.length === 0,
            info,
        };
    } catch (error) {
        let errorMessage = "An unknown error occurred during email transport.";
        let errorCode: string | undefined;
        // Consider logging error.stack in non-production or if it's deemed safe and useful.
        // let errorStack: string | undefined;

        if (error instanceof Error) {
            errorMessage = error.message;
            // errorStack = error.stack;
            // Attempt to get an error code if it exists (common in network/nodemailer errors)
            const potentialCodeFromError = (error as unknown as Record<string, unknown>).code;
            if (typeof potentialCodeFromError === "string") {
                errorCode = potentialCodeFromError;
            }
        } else if (typeof error === "object" && error !== null && "message" in error) {
            errorMessage = String((error as { message: unknown }).message);
            const potentialCodeFromObject = (error as unknown as Record<string, unknown>).code;
            if (typeof potentialCodeFromObject === "string") {
                errorCode = potentialCodeFromObject;
            }
        } else if (typeof error === "string") {
            errorMessage = error;
        }

        logger.error("Failed to process email job", {
            trace: "email.job.processFailed",
            jobId: job.id,
            errorDetails: { // Nesting under 'errorDetails' to make it clear
                message: errorMessage,
                code: errorCode,
                isSetupPhaseFailure: transporter === null, // Hint if error was likely during setup
                // stack: errorStack, // (Optional)
            },
        });
        return {
            success: false,
        };
    }
}

