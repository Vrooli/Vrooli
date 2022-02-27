import { CODE, emailCreate, emailUpdate } from "@local/shared";
import { CustomError } from "../error";
import { Count, Email, EmailCreateInput, EmailUpdateInput } from "../schema/types";
import { PrismaType } from "types";
import { addSupplementalFields, CUDInput, CUDResult, FormatConverter, modelToGraphQL, relationshipToPrisma, RelationshipTypes, selectHelper, ValidateMutationsInput } from "./base";
import { hasProfanity } from "../utils/censor";
import { profileValidater } from "./profile";

//==============================================================
/* #region Custom Components */
//==============================================================

export const emailFormatter = (): FormatConverter<Email> => ({})

export const emailVerifier = () => ({
    profanityCheck(data: EmailCreateInput): void {
        if (hasProfanity(data.emailAddress)) throw new CustomError(CODE.BannedWord);
    },
})

export const emailMutater = (prisma: PrismaType, verifier: any) => ({
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by emails (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'emails', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] });
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: createMany as EmailCreateInput[],
            updateMany: updateMany as EmailUpdateInput[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<EmailCreateInput, EmailUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            // Make sure emails aren't already in use
            const emails = await prisma.email.findMany({
                where: { emailAddress: { in: createMany.map(email => email.emailAddress) } },
            });
            if (emails.length > 0) throw new CustomError(CODE.EmailInUse);
            // Perform other checks
            for (const email of createMany) {
                // Check for valid arguments
                emailCreate.validateSync(email, { abortEarly: false });
                // Check for censored words
                verifier.profanityCheck(email as EmailCreateInput);
            }
        }
        if (updateMany) {
            // Make sure emails are owned by user
            const emails = await prisma.email.findMany({
                where: {
                    AND: [
                        { id: { in: updateMany.map(email => email.id) } },
                        { userId },
                    ],
                },
            });
            if (emails.length !== updateMany.length) throw new CustomError(CODE.EmailInUse, 'At least one of these emails is not yours');
            for (const email of updateMany) {
                // Check for valid arguments
                emailUpdate.validateSync(email, { abortEarly: false });
            }
        }
    },
    async cud({ info, userId, createMany, updateMany, deleteMany }: CUDInput<EmailCreateInput, EmailUpdateInput>): Promise<CUDResult<Email>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Check for existing email
                const existing = await prisma.email.findUnique({ where: { emailAddress: input.emailAddress } });
                if (existing) throw new CustomError(CODE.EmailInUse)
                // Create object
                const currCreated = await prisma.email.create({
                    data: {
                        userId,
                        ...input
                    },
                    ...selectHelper(info)
                });
                // Send verification email
                await profileValidater().setupVerificationCode(input.emailAddress, prisma);
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, info);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.email.findFirst({
                    where: {
                        AND: [
                            { id: input.id },
                            { userId },
                        ]
                    }
                })
                if (!object) throw new CustomError(CODE.NotFound, "Email not found");
                // Update
                object = await prisma.email.update({
                    where: { id: object.id },
                    data: input,
                    ...selectHelper(info)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(object, info);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            // Find
            const emails = await prisma.email.findMany({
                where: {
                    AND: [
                        { id: { in: deleteMany } },
                        { userId },
                    ]
                },
                select: {
                    id: true,
                    verified: true,
                }
            })
            if (!emails) throw new CustomError(CODE.NotFound, "Email not found");
            // Check if user has at least one verified authentication method, besides the one being deleted
            const numberOfVerifiedEmailDeletes = emails.filter(email => email.verified).length;
            const verifiedEmailsCount = await prisma.email.count({
                where: { userId, verified: true }
            })
            const verifiedWalletsCount = await prisma.wallet.count({
                where: { userId, verified: true }
            })
            const wontHaveVerifiedEmail = numberOfVerifiedEmailDeletes >= verifiedEmailsCount;
            const wontHaveVerifiedWallet = verifiedWalletsCount <= 0;
            if (wontHaveVerifiedEmail || wontHaveVerifiedWallet) throw new CustomError(CODE.InternalError, "Must leave at least one verified authentication method");
            // Delete
            deleted = await prisma.email.deleteMany({
                where: { id: { in: deleteMany } },
            });
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function EmailModel(prisma: PrismaType) {
    const prismaObject = prisma.email;
    const format = emailFormatter();
    const verify = emailVerifier();
    const mutate = emailMutater(prisma, verify);

    return {
        prismaObject,
        ...format,
        ...verify,
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================