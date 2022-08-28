import { CODE } from "@shared/consts";
import { emailsCreate, emailsUpdate } from "@shared/validation";
import { CustomError } from "../../error";
import { Count, Email, EmailCreateInput, EmailUpdateInput } from "../../schema/types";
import { PrismaType } from "../../types";
import { CUDInput, CUDResult, FormatConverter, modelToGraphQL, relationshipToPrisma, RelationshipTypes, selectHelper, ValidateMutationsInput } from "./base";
import { validateProfanity } from "../../utils/censor";
import { genErrorCode } from "../../logger";
import { ProfileModel } from "./profile";

//==============================================================
/* #region Custom Components */
//==============================================================

export const emailFormatter = (): FormatConverter<Email, any> => ({
    relationshipMap: {
        '__typename': 'Email',
        'user': 'Profile',
    }
})

export const emailVerifier = () => ({
    profanityCheck(data: EmailCreateInput[]): void {
        validateProfanity(data.map(d => d.emailAddress));
    },
})

export const emailMutater = (prisma: PrismaType) => ({
    toDBShapeAdd(userId: string | null, data: EmailCreateInput): any {
        return {
            userId,
            emailAddress: data.emailAddress,
            receivesAccountUpdates: data.receivesAccountUpdates ?? true,
            receivesBusinessUpdates: data.receivesBusinessUpdates ?? true,
        }
    },
    toDBShapeUpdate(userId: string | null, data: EmailUpdateInput): any {
        return {
            id: data.id,
            receivesAccountUpdates: data.receivesAccountUpdates ?? undefined,
            receivesBusinessUpdates: data.receivesBusinessUpdates ?? undefined,
        }
    },
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'emails',
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by emails (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] });
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: createMany as EmailCreateInput[],
            updateMany: updateMany as { where: { id: string }, data: EmailUpdateInput }[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<EmailCreateInput, EmailUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId)
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0043') });
        if (createMany) {
            emailsCreate.validateSync(createMany, { abortEarly: false });
            emailVerifier().profanityCheck(createMany);
            // Make sure emails aren't already in use
            const emails = await prisma.email.findMany({
                where: { emailAddress: { in: createMany.map(email => email.emailAddress) } },
            });
            if (emails.length > 0)
                throw new CustomError(CODE.EmailInUse, 'Email address is already in use', { code: genErrorCode('0044') });
        }
        if (updateMany) {
            emailsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            // Make sure emails are owned by user
            const emails = await prisma.email.findMany({
                where: {
                    AND: [
                        { id: { in: updateMany.map(email => email.where.id) } },
                        { userId },
                    ],
                },
            });
            if (emails.length !== updateMany.length)
                throw new CustomError(CODE.EmailInUse, 'At least one of these emails is not yours', { code: genErrorCode('0045') });
        }
    },
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<EmailCreateInput, EmailUpdateInput>): Promise<CUDResult<Email>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Check for existing email
                const existing = await prisma.email.findUnique({ where: { emailAddress: input.emailAddress } });
                if (existing)
                    throw new CustomError(CODE.EmailInUse, 'Email address is already in use', { code: genErrorCode('0046') });
                // Create object
                const currCreated = await prisma.email.create({
                    data: this.toDBShapeAdd(userId, input),
                    ...selectHelper(partialInfo)
                });
                // Send verification email
                await ProfileModel.verify.setupVerificationCode(input.emailAddress, prisma);
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
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
                            input.where,
                            { userId },
                        ]
                    }
                })
                if (!object)
                    throw new CustomError(CODE.NotFound, "Email not found", { code: genErrorCode('0047') });
                // Update
                object = await prisma.email.update({
                    where: input.where,
                    data: this.toDBShapeUpdate(userId, input.data),
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(object, partialInfo);
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
            if (!emails)
                throw new CustomError(CODE.NotFound, "Email not found", { code: genErrorCode('0048') });
            // Check if user has at least one verified authentication method, besides the one being deleted
            const numberOfVerifiedEmailDeletes = emails.filter(email => email.verified).length;
            const verifiedEmailsCount = await prisma.email.count({
                where: { userId, verified: true }//TODO or organizationId
            })
            const verifiedWalletsCount = await prisma.wallet.count({
                where: { userId, verified: true }//TODO or organizationId
            })
            const wontHaveVerifiedEmail = numberOfVerifiedEmailDeletes >= verifiedEmailsCount;
            const wontHaveVerifiedWallet = verifiedWalletsCount <= 0;
            if (wontHaveVerifiedEmail || wontHaveVerifiedWallet)
                throw new CustomError(CODE.InternalError, "Cannot delete all verified authentication methods", { code: genErrorCode('0049') });
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

export const EmailModel = ({
    prismaObject: (prisma: PrismaType) => prisma.email,
    format: emailFormatter(),
    mutate: emailMutater,
    verify: emailVerifier(),
})

//==============================================================
/* #endregion Model */
//==============================================================