import { Email, EmailCreateInput, MaxObjects } from '@shared/consts';
import { PrismaType } from "../types";
import { CustomError, Trigger } from "../events";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { defaultPermissions } from '../utils';
import { emailValidation } from '@shared/validation';

const __typename = 'Email' as const;
const suppFields = [] as const;
export const EmailModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: EmailCreateInput,
    GqlUpdate: undefined,
    GqlModel: Email,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.emailUpsertArgs['create'],
    PrismaUpdate: Prisma.emailUpsertArgs['update'],
    PrismaModel: Prisma.emailGetPayload<SelectWrap<Prisma.emailSelect>>,
    PrismaSelect: Prisma.emailSelect,
    PrismaWhere: Prisma.emailWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.email,
    display: {
        select: () => ({ id: true, emailAddress: true }),
        label: (select) => select.emailAddress,
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            user: 'User',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, userData }) => ({
                emailAddress: data.emailAddress,
                user: { connect: { id: userData.id } },
            }),
        },
        trigger: {
            onCreated: ({ prisma, userData }) => {
                Trigger(prisma, userData.languages).createEmail(userData.id)
            },
        },
        yup: emailValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            user: 'User',
        }),
        permissionResolvers: defaultPermissions,
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
        },
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ user: { id: userId } }),
        }
    },
})