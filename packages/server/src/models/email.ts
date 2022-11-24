import { emailsCreate, emailsUpdate } from "@shared/validation";
import { Email, EmailCreateInput, EmailUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { CustomError, Trigger } from "../events";
import { Formatter, GraphQLModelType, Validator, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { UserModel } from "./user";

const formatter = (): Formatter<Email, any> => ({
    relationshipMap: {
        __typename: 'Email',
        user: 'Profile',
    }
})

const validator = (): Validator<
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
    permissionsSelect: (...params) => ({ id: true, user: { select: UserModel.validate.permissionsSelect(...params) } }),
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
        create: async ({ createMany, prisma, userData }) => {
            // Prevent creating emails if at least one is already in use
            const existingEmails = await prisma.email.findMany({
                where: { emailAddress: { in: createMany.map(x => x.emailAddress) } },
            });
            if (existingEmails.length > 0) throw new CustomError('0044', 'EmailInUse', userData.languages)
        },
        delete: async ({ deleteMany, prisma, userData }) => {
            // Prevent deleting emails if it will leave you with less than one verified authentication method
            const allEmails = await prisma.email.findMany({
                where: { user: { id: userData.id } },
                select: { id: true, verified: true }
            });
            const remainingVerifiedEmailsCount = allEmails.filter(x => !deleteMany.includes(x.id) && x.verified).length;
            const verifiedWalletsCount = await prisma.wallet.count({
                where: { user: { id: userData.id }, verified: true },
            });
            if (remainingVerifiedEmailsCount + verifiedWalletsCount < 1)
                throw new CustomError('0049', 'MustLeaveVerificationMethod', userData.languages);
        }
    }
})

const mutater = (): Mutater<
    Email,
    { graphql: EmailCreateInput, db: Prisma.emailUpsertArgs['create'] },
    { graphql: EmailUpdateInput, db: Prisma.emailUpsertArgs['update'] },
    { graphql: EmailCreateInput, db: Prisma.emailCreateWithoutUserInput },
    { graphql: EmailUpdateInput, db: Prisma.emailUpdateWithoutUserInput }
> => ({
    shape: {
        create: async ({ data, userData }) => {
            return {
                userId: userData.id,
                emailAddress: data.emailAddress,
            }
        },
        update: async ({ data }) => {
            return {
                id: data.id,
            }
        },
        relCreate: mutater().shape.create,
        relUpdate: mutater().shape.update,
    },
    trigger: {
        onCreated: ({ created, prisma, userData }) => {
            for (const c of created) {
                Trigger(prisma, userData.languages).objectCreate('Email', c.id as string, userData.id);
            }
        },
    },
    yup: { create: emailsCreate, update: emailsUpdate },
})

export const EmailModel = ({
    delegate: (prisma: PrismaType) => prisma.email,
    format: formatter(),
    mutate: mutater(),
    type: 'Email' as GraphQLModelType,
    validate: validator(),
})