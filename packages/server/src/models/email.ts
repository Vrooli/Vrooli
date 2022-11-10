import { CODE } from "@shared/consts";
import { emailsCreate, emailsUpdate } from "@shared/validation";
import { Email, EmailCreateInput, EmailUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { relationshipToPrisma, RelationshipTypes } from "./builder";
import { CustomError, genErrorCode } from "../events";
import { FormatConverter, CUDInput, CUDResult, GraphQLModelType, Permissioner } from "./types";
import { cudHelper } from "./actions";
import { Prisma } from "@prisma/client";

export const emailFormatter = (): FormatConverter<Email, any> => ({
    relationshipMap: {
        '__typename': 'Email',
        'user': 'Profile',
    }
})

export const emailPermissioner = (): Permissioner<any, any> => ({
    async get() {
        return [] as any;
    },
    ownershipQuery: (userId) => ({
        user: { id: userId }
    }),
})

export const emailValidator = () => ({
    // Profanity fields to check in addition to translated fields
    additionalProfanityFields: ['emailAddress'],
    // Prevent creating emails if at least one is already in use
    preventCreateIf: [{
        query: (createMany: EmailCreateInput[]) => ({ emailAddress: { in: createMany.map(email => email.emailAddress) } }),
        error: () => new CustomError(CODE.EmailInUse, 'Email address is already in use', { code: genErrorCode('0044') })
    }],
    // Prevent deleting emails if it will leave you with less than one 
    // verified authentification method
    preventDeleteIf: [{
    }],
    // // Check if user has at least one verified authentication method, besides the one being deleted
    // const numberOfVerifiedEmailDeletes = emails.filter(email => email.verified).length;
    // const verifiedEmailsCount = await prisma.email.count({
    //     where: { userId, verified: true }//TODO or organizationId
    // })
    // const verifiedWalletsCount = await prisma.wallet.count({
    //     where: { userId, verified: true }//TODO or organizationId
    // })
    // const wontHaveVerifiedEmail = numberOfVerifiedEmailDeletes >= verifiedEmailsCount;
    // const wontHaveVerifiedWallet = verifiedWalletsCount <= 0;
    // if (wontHaveVerifiedEmail || wontHaveVerifiedWallet)
    //     throw new CustomError(CODE.InternalError, "Cannot delete all verified authentication methods", { code: genErrorCode('0049') });
})

export const emailMutater = (prisma: PrismaType) => ({
    shapeCreate(userId: string, data: EmailCreateInput): Prisma.emailUpsertArgs['create'] {
        return {
            userId,
            emailAddress: data.emailAddress,
        }
    },
    shapeUpdate(userId: string, data: EmailUpdateInput): Prisma.emailUpsertArgs['update'] {
        return {
            id: data.id,
        }
    },
    async relationshipBuilder(
        userId: string,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'emails',
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by emails (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] });
        if (Array.isArray(formattedInput.create)) {
            formattedInput.create = formattedInput.create.map(d => this.shapeCreate(userId, d as any));
        }
        if (Array.isArray(formattedInput.update)) {
            formattedInput.update = formattedInput.update.map(d => ({
                where: d.where,
                data: this.shapeUpdate(userId, d.data as any),
            }));
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    cud(params: CUDInput<EmailCreateInput, EmailUpdateInput>): Promise<CUDResult<Email>> {
        return cudHelper({
            ...params,
            objectType: 'Email',
            prisma,
            prismaObject: (p) => p.email,
            yup: { yupCreate: emailsCreate, yupUpdate: emailsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const EmailModel = ({
    prismaObject: (prisma: PrismaType) => prisma.email,
    format: emailFormatter(),
    mutate: emailMutater,
    permissions: emailPermissioner,
    type: 'Email' as GraphQLModelType,
    validate: emailValidator,
})