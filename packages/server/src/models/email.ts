import { CODE } from "@shared/consts";
import { emailsCreate, emailsUpdate } from "@shared/validation";
import { Email, EmailCreateInput, EmailUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { relationshipBuilderHelper } from "./builder";
import { CustomError, genErrorCode } from "../events";
import { FormatConverter, CUDInput, CUDResult, GraphQLModelType, Validator, ModelLogic } from "./types";
import { cudHelper } from "./actions";
import { Prisma } from "@prisma/client";

export const emailFormatter = (): FormatConverter<Email, any> => ({
    relationshipMap: {
        __typename: 'Email',
        user: 'Profile',
    }
})

export const emailValidator = (): Validator<EmailCreateInput, EmailUpdateInput, Email, any, Prisma.emailSelect, Prisma.emailWhereInput> => ({
    validateMap: {
        __typename: 'Email',
        user: 'User',
    },
    permissionsSelect: { user: { select: { id: true } } },
    permissionsFromSelect: (data) => dafdsafdasfdta as any,
    ownerOrMemberWhere: (userId) => ({ user: { id: userId } }),
    profanityFields: ['emailAddress'],
    validations: {
        create: async (createMany, prisma) => {
            // Prevent creating emails if at least one is already in use
            const existingEmails = await prisma.email.findMany({
                where: { emailAddress: { in: createMany.map(x => x.emailAddress) } },
            });
            if (existingEmails.length > 0) throw new CustomError(CODE.EmailInUse, 'Email address is already in use', { code: genErrorCode('0044') })
        },
        delete: async (deleteMany, prisma, userId) => {
            // Prevent deleting emails if it will leave you with less than one verified authentication method
            const allEmails = await prisma.email.findMany({
                where: { user: { id: userId } },
                select: { id: true, verified: true }
            });
            const remainingVerifiedEmailsCount = allEmails.filter(x => !deleteMany.includes(x.id) && x.verified).length;
            const verifiedWalletsCount = await prisma.wallet.count({
                where: { user: { id: userId }, verified: true },
            });
            if (remainingVerifiedEmailsCount + verifiedWalletsCount < 1)
                throw new CustomError(CODE.InternalError, "Cannot delete all verified authentication methods", { code: genErrorCode('0049') });
        }
    }
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
        data: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'emails',
    ): Promise<{ [x: string]: any } | undefined> {
        return relationshipBuilderHelper({
            data,
            relationshipName,
            isAdd,
            isTransferable: false,
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            userId,
        });
    },
    cud(params: CUDInput<EmailCreateInput, EmailUpdateInput>): Promise<CUDResult<Email>> {
        return cudHelper({
            ...params,
            objectType: 'Email',
            prisma,
            yup: { yupCreate: emailsCreate, yupUpdate: emailsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const EmailModel = ({
    prismaObject: (prisma: PrismaType) => prisma.email,
    format: emailFormatter(),
    mutate: emailMutater,
    type: 'Email' as GraphQLModelType,
    validate: emailValidator(),
})