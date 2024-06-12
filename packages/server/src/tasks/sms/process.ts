import { Job } from "bull";
import { Twilio } from "twilio";
import { logger } from "../../events/logger";
import { SmsProcessPayload } from "./queue";

let texting_client: Twilio | null = null;
let phoneNumber: string | null = null;

/**
 * Function to setup twilio client. This is needed because 
 * the auth token is loaded from the secrets location, so it's 
 * not available at startup.
 */
export const setupTextingClient = async () => {
    if (texting_client === null) {
        try {
            const client = await import("twilio");
            texting_client = client.default(process.env.TWILIO_ACCOUNT_SID, process.env.TWILIO_AUTH_TOKEN);
        } catch (error) {
            logger.warning("TWILIO client could not be initialized. Sending SMS will not work", { trace: "0013", error });
        }
    }
    if (phoneNumber === null) {
        if (process.env.TWILIO_PHONE_NUMBER) {
            phoneNumber = process.env.TWILIO_PHONE_NUMBER;
        } else {
            logger.warning("TWILIO phone number not set. Sending SMS will not work", { trace: "0015" });
        }
    }
};

export const smsProcess = async (job: Job<SmsProcessPayload>) => {
    try {
        await setupTextingClient();
        if (texting_client === null || phoneNumber === null) {
            logger.error("Cannot send SMS. Texting client not initialized", { trace: "0014" });
            return false;
        }
        const messagePromises = job.data.to.map(t => {
            return texting_client!.messages.create({
                to: t,
                from: phoneNumber!,
                body: job.data.body,
            });
        });
        const results = await Promise.all(messagePromises);
        results.forEach(result => console.log(result));
        return true;
    } catch (err) {
        logger.error("Error sending sms", { trace: "0082" });
    }
    return false;
};

