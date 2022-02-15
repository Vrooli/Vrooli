import { CODE, emailCreate, emailUpdate } from "@local/shared";
import { CustomError } from "../error";
import { GraphQLResolveInfo } from "graphql";
import { DeleteOneInput, Email, EmailCreateInput, EmailUpdateInput, Success } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { MODEL_TYPES, relationshipToPrisma } from "./base";
import bcrypt from 'bcrypt';
import { sendVerificationLink } from "../worker/email/queue";
import { hasProfanity } from "../utils/censor";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Handles the authorized adding, updating, and deleting of emails.
 * A user can only add/update/delete their own email, and must leave at least
 * one authentication method (either an email or wallet).
 */
const emailer = (prisma: PrismaType) => ({
    /**
    * Sends a verification email to the user's email address.
    */
    async setupVerificationCode(emailAddress: string): Promise<void> {
        // Generate new code
        const verificationCode = bcrypt.genSaltSync(8).replace('/', '')
        // Store code and request time in email row
        const email = await prisma.email.update({
            where: { emailAddress },
            data: { verificationCode, lastVerificationCodeRequestAttempt: new Date().toISOString() },
            select: { userId: true }
        })
        // If email is not associated with a user, throw error
        if (!email.userId) throw new CustomError(CODE.ErrorUnknown, 'Email not associated with a user');
        // Send new verification email
        sendVerificationLink(emailAddress, email.userId, verificationCode);
        // TODO send email to existing emails from user, warning of new email
    },
    /**
     * Add, update, or remove an email relationship from your profile
     */
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by emails (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma(input, 'emails', isAdd, [], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            // Make sure emails aren't already in use
            const emails = await prisma.email.findMany({
                where: { emailAddress: { in: formattedInput.create.map(email => email.emailAddress) } },
            });
            if (emails.length > 0) throw new CustomError(CODE.EmailInUse);
            // Perform other checks
            for (const email of formattedInput.create) {
                // Check for valid arguments
                emailCreate.validateSync(input, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(email.emailAddress)) throw new CustomError(CODE.BannedWord);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            // Make sure emails are owned by user
            const emails = await prisma.email.findMany({
                where: {
                    AND: [
                        { emailAddress: { in: formattedInput.update.map(email => email.emailAddress) } },
                        { userId },
                    ],
                },
            });
            if (emails.length !== formattedInput.update.length) throw new CustomError(CODE.EmailInUse, 'At least one of these emails is not yours');
            for (const email of formattedInput.update) {
                // Check for valid arguments
                emailUpdate.validateSync(input, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(email.emailAddress)) throw new CustomError(CODE.BannedWord);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async create(
        userId: string,
        input: EmailCreateInput,
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<Email>> {
        // Check for valid arguments
        emailCreate.validateSync(input, { abortEarly: false });
        // Check for existing email
        const existing = await prisma.email.findUnique({ where: { emailAddress: input.emailAddress } });
        if (existing) throw new CustomError(CODE.EmailInUse)
        // Add email
        let email = await prisma.email.create({
            data: {
                userId,
                ...input
            } as any,
        });
        // Send verification email
        await this.setupVerificationCode(email.emailAddress);
        // Return email
        return email;
    },
    async update(
        userId: string,
        input: EmailUpdateInput,
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<Email>> {
        // Check for valid arguments
        emailUpdate.validateSync(input, { abortEarly: false });
        // Find email
        let email = await prisma.email.findFirst({
            where: {
                AND: [
                    { id: input.id },
                    { userId },
                ]
            }
        })
        if (!email) throw new CustomError(CODE.NotFound, "Email not found");
        // Update email
        email = await prisma.email.update({
            where: { id: email.id },
            data: info as any,
        });
        // Return email
        return email;
    },
    async delete(userId: string, input: DeleteOneInput): Promise<Success> {
        // Find
        const email = await prisma.email.findFirst({
            where: {
                AND: [
                    { id: input.id },
                    { userId },
                ]
            }
        })
        if (!email) throw new CustomError(CODE.NotFound, "Email not found");
        // Check if user has at least one verified authentication method, besides the one being deleted
        const verifiedEmailsCount = await prisma.email.count({
            where: {
                userId,
                verified: true,
            }
        })
        const verifiedWalletsCount = await prisma.wallet.count({
            where: {
                userId,
                verified: true,
            }
        })
        const wontHaveVerifiedEmail = email.verified ? verifiedEmailsCount <= 1 : verifiedEmailsCount <= 0;
        const wontHaveVerifiedWallet = verifiedWalletsCount <= 0;
        if (wontHaveVerifiedEmail || wontHaveVerifiedWallet) throw new CustomError(CODE.InternalError, "Must leave at least one verified authentication method");
        // Delete
        await prisma.email.delete({
            where: { id: email.id },
        });
        return { success: true };
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function EmailModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Email;

    return {
        prisma,
        model,
        ...emailer(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================