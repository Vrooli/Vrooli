import { logger } from "../../events/logger";

let texting_client: any = null;

/**
 * Function to setup twilio client. This is needed because 
 * the auth token is loaded from the secrets location, so it's 
 * not available at startup.
 */
export const setupTextingClient = () => {
    if (texting_client === null) {
        try {
            texting_client = require("twilio")(process.env.TWILIO_ACCOUNT_SID, process.env.TWILIO_AUTH_TOKEN);
        } catch (error) {
            logger.warning("TWILIO client could not be initialized. Sending SMS will not work", { trace: "0013", error });
        }
    }
};

export async function smsProcess(job: any) {
    setupTextingClient();
    if (texting_client === null) {
        logger.error("Cannot send SMS. Texting client not initialized", { trace: "0014" });
        return false;
    }
    job.data.to.forEach((t: any) => {
        texting_client.messages.create({
            to: t,
            from: process.env.PHONE_NUMBER,
            body: job.data.body,
        }).then((message: any) => console.log(message));
    });
    return true;
}
