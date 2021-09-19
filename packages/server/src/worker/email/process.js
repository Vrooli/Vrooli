import nodemailer from 'nodemailer';

const HOST = 'smtp.gmail.com';
const PORT = 465;
const transporter = nodemailer.createTransport({
    host: HOST,
    port: PORT,
    auth: {
        user: process.env.SITE_EMAIL_USERNAME,
        pass: process.env.SITE_EMAIL_PASSWORD
    }
})

export async function emailProcess(job) {
    transporter.sendMail({
        from : `"${process.env.SITE_EMAIL_FROM}" <${process.env.SITE_EMAIL_USERNAME}>`,
        to: job.data.to.join(', '),
        subject: job.data.subject,
        text: job.data.text,
        html: job.data.html
    }).then(info => {
        return {
            'success': info.rejected.length === 0,
            'info': info
        }
    }).catch(err => {
        console.error(err);
        return {
            'success': false
        }
    });
}