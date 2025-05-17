import { Job } from "bullmq";
import nodemailer from "nodemailer";
import { logger } from "../../events/logger.js";
import { EmailTask } from "../taskTypes.js";

const HOST = "smtp.gmail.com";
const PORT = 465;

let transporter: nodemailer.Transporter | null = null;

/**
 * Function to setup transporter. This is needed because 
 * the password is loaded from the secrets location, so it's 
 * not available at startup.
 */
export function setupTransporter() {
    if (transporter === null) {
        transporter = nodemailer.createTransport({
            host: HOST,
            port: PORT,
            secure: true,
            auth: {
                user: process.env.SITE_EMAIL_USERNAME,
                pass: process.env.SITE_EMAIL_PASSWORD,
            },
        });
    }
}

/**
 * Process an email job from the queue
 * 
 * @param job The email job to process
 * @returns Result with success indicator and email info
 */
export async function emailProcess(job: Job<EmailTask>) {
    setupTransporter();

    if (!transporter) {
        logger.error("Email transporter not initialized", { jobId: job.id });
        return { success: false };
    }

    try {
        const info = await transporter.sendMail({
            from: `"${process.env.SITE_EMAIL_FROM}" <${process.env.SITE_EMAIL_ALIAS ?? process.env.SITE_EMAIL_USERNAME}>`,
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
        logger.error("Caught error using email transporter", { trace: "0012", error, jobId: job.id });
        return {
            success: false,
        };
    }
}

