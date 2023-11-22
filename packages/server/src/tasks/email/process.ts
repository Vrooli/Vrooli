import { Job } from "bull";
import nodemailer from "nodemailer";
import { logger } from "../../events/logger";
import { EmailProcessPayload } from "./queue";

const HOST = "smtp.gmail.com";
const PORT = 465;

let transporter: nodemailer.Transporter | null = null;

/**
 * Function to setup transporter. This is needed because 
 * the password is loaded from the secrets location, so it's 
 * not available at startup.
 */
export const setupTransporter = () => {
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
};

export async function emailProcess(job: Job<EmailProcessPayload>) {
    setupTransporter();
    transporter!.sendMail({
        from: `"${process.env.SITE_EMAIL_FROM}" <${process.env.SITE_EMAIL_ALIAS ?? process.env.SITE_EMAIL_USERNAME}>`,
        to: job.data.to.join(", "),
        subject: job.data.subject,
        text: job.data.text,
        html: job.data.html,
    }).then((info: any) => {
        return {
            "success": info.rejected.length === 0,
            info,
        };
    }).catch((error: any) => {
        logger.error("Caught error using email transporter", { trace: "0012", error });
        return {
            "success": false,
        };
    });
}
