import { type Job } from "bullmq";
import * as webpush from "web-push";
import { logger } from "../../events/logger.js";
import { type PushNotificationTask } from "../taskTypes.js";

let vapidDetailsSet = false;
// Flag to track if critical setup failed (e.g., missing VAPID keys)
let vapidCriticalSetupFailed = false;

/**
 * Function to set VAPID details. This is needed because
 * the private key is loaded from the secrets location, so it's
 * not available at startup.
 * This function now includes robust checks for VAPID keys.
 */
export function setVapidDetails() {
    const vapidPublicKey = process.env.VAPID_PUBLIC_KEY;
    const vapidPrivateKey = process.env.VAPID_PRIVATE_KEY;
    // Use site email variables as fallback chain
    const contactEmail = process.env.SITE_EMAIL_ALIAS || process.env.SITE_EMAIL_USERNAME;

    // Check for missing VAPID keys or contact email
    if (!vapidPublicKey || !vapidPrivateKey || !contactEmail) {
        if (!vapidCriticalSetupFailed) { // Log detailed error only on first detection
            const missingVars: string[] = [];
            if (!vapidPublicKey) missingVars.push("VAPID_PUBLIC_KEY");
            if (!vapidPrivateKey) missingVars.push("VAPID_PRIVATE_KEY");
            if (!contactEmail) missingVars.push("SITE_EMAIL_ALIAS or SITE_EMAIL_USERNAME");
            logger.error(
                `VAPID setup failed: ${missingVars.join(", ")} is not defined in environment variables. Push notifications cannot be configured.`,
                { trace: "push.vapid.initFailedKeysMissing" },
            );
        }
        vapidCriticalSetupFailed = true;
        vapidDetailsSet = false; // Ensure details are not marked as set
        return;
    }

    // VAPID keys AND contact email are present.
    // If criticalSetupFailed was true, it means keys were missing but are now available.
    if (vapidCriticalSetupFailed) {
        logger.info(
            "VAPID keys are now available. Attempting to reinitialize VAPID details.",
            { trace: "push.vapid.keysRestored" },
        );
        vapidCriticalSetupFailed = false; // Reset flag
        vapidDetailsSet = false; // Allow re-initialization
    }

    // If already successfully set and no critical failure, do nothing.
    if (vapidDetailsSet) {
        return;
    }

    try {
        webpush.setVapidDetails(
            `mailto:${contactEmail}`, // Use the validated contact email
            vapidPublicKey,
            vapidPrivateKey,
        );
        vapidDetailsSet = true;
        vapidCriticalSetupFailed = false; // Explicitly false on success
        logger.info("VAPID details initialized successfully.", { trace: "push.vapid.initialized" });
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        logger.error(
            "Failed to set VAPID details using webpush.setVapidDetails. Push notifications may not work.",
            {
                trace: "push.vapid.setVapidDetailsFailed",
                error: errorMessage,
            },
        );
        vapidDetailsSet = false;
        // We don't set vapidCriticalSetupFailed to true here as keys were present,
        // but the library call failed for other reasons (e.g. invalid keys).
    }
}

export async function pushProcess({ data, id: jobId }: Job<PushNotificationTask>) {
    try {
        const { subscription, payload } = data;
        setVapidDetails();

        // Check if VAPID setup is complete and not in a critical failure state
        if (!vapidDetailsSet || vapidCriticalSetupFailed) {
            const errorMessage = "Cannot send push notification: VAPID details are not properly configured or setup failed.";
            logger.error(
                errorMessage,
                {
                    jobId,
                    trace: "push.send.precheckFailed",
                    vapidDetailsSet,
                    vapidCriticalSetupFailed,
                },
            );
            throw new Error(errorMessage);
        }

        await webpush.sendNotification(subscription, JSON.stringify(payload));
        logger.info("Push notification sent successfully.", { jobId, trace: "push.send.success" });
    } catch (err) {
        let errorMessage = "An unknown error occurred during push notification processing.";
        let errorCode: string | undefined;
        // const errorStack = err instanceof Error ? err.stack : undefined;

        if (err instanceof Error) {
            errorMessage = err.message;
            // Attempt to get an error code if it exists (common in web-push errors)
            const potentialCodeFromError = (err as { statusCode?: number, code?: string }).code ?? (err as { statusCode?: number, code?: string }).statusCode?.toString();
            if (typeof potentialCodeFromError === "string" || typeof potentialCodeFromError === "number") {
                errorCode = String(potentialCodeFromError);
            }
        } else if (typeof err === "object" && err !== null && "message" in err) {
            errorMessage = String((err as { message: unknown }).message);
        } else if (typeof err === "string") {
            errorMessage = err;
        }

        logger.error("Error sending push notification", {
            jobId,
            trace: "push.send.failed", // Consistent trace for send failures
            errorDetails: {
                message: errorMessage,
                code: errorCode, // web-push errors often have a statusCode or code
                // stack: errorStack, // Optional: for more detailed debugging
            },
        });
        // Depending on queue retry strategy, you might want to throw err here
        // to mark the job as failed for retry.
        throw err;
    }
}
