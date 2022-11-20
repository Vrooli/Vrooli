import { emailsCreate, emailsUpdate } from "@shared/validation";
import { Email, EmailCreateInput, EmailUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { relationshipBuilderHelper } from "./builder";
import { CustomError, Trigger } from "../events";
import { FormatConverter, CUDInput, CUDResult, GraphQLModelType, Validator, Mutater } from "./types";
import { cudHelper } from "./actions";
import { Prisma } from "@prisma/client";
import { userValidator } from "./user";

export const emailFormatter = (): FormatConverter<Email, any> => ({
    relationshipMap: {
        __typename: 'Email',
        user: 'Profile',
    }
})

export const emailValidator = (): Validator<
    EmailCreateInput,
    EmailUpdateInput,
    Email,
    Prisma.emailGetPayload<{ select: { [K in keyof Required<Prisma.emailSelect>]: true } }>,
    any,
    Prisma.emailSelect,
    Prisma.emailWhereInput
> => ({
    validateMap: {
        __typename: 'Email',
        user: 'User',
    },
    permissionsSelect: (userId) => ({ id: true, user: { select: userValidator().permissionsSelect(userId) } }),
    permissionResolvers: ({ isAdmin }) => ([
        ['canDelete', async () => isAdmin],
        ['canEdit', async () => isAdmin],
    ]),
    ownerOrMemberWhere: (userId) => ({ user: { id: userId } }),
    owner: (data) => ({
        User: data.user,
    }),
    isDeleted: () => false,
    isPublic: () => false,
    profanityFields: ['emailAddress'],
    validations: {
        create: async (createMany, prisma) => {
            // Prevent creating emails if at least one is already in use
            const existingEmails = await prisma.email.findMany({
                where: { emailAddress: { in: createMany.map(x => x.emailAddress) } },
            });
            if (existingEmails.length > 0) throw new CustomError('EmailInUse', { trace: '0044' })
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
                throw new CustomError('MustLeaveVerificationMethod', { trace: '0049' });
        }
    }
})

export const emailMutater = (prisma: PrismaType): Mutater<Email> => ({
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
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            onCreated: (created) => {
                for (const c of created) {
                    Trigger(prisma).objectCreate('Email', c.id as string, params.userData.id);
                }
            },
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