import { emailsCreate } from "@shared/validation";
import { Email, EmailCreateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { CustomError, Trigger } from "../events";
import { Formatter, Validator, Mutater, Displayer } from "./types";
import { Prisma } from "@prisma/client";
import { UserModel } from "./user";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: EmailCreateInput,
    GqlModel: Email,
    GqlPermission: any,
    PrismaCreate: Prisma.emailUpsertArgs['create'],
    PrismaUpdate: Prisma.emailUpsertArgs['update'],
    PrismaModel: Prisma.emailGetPayload<SelectWrap<Prisma.emailSelect>>,
    PrismaSelect: Prisma.emailSelect,
    PrismaWhere: Prisma.emailWhereInput,
}

const __typename = 'Email' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
    }
})

const validator = (): Validator<Model> => ({
    validateMap: {
        __typename: 'Email',
        user: 'User',
    },
    isTransferable: false,
    maxObjects: {
        User: {
            private: 5,
            public: 1,
        },
        Organization: 0,
    },
    permissionsSelect: (...params) => ({ id: true, user: { select: UserModel.validate.permissionsSelect(...params) } }),
    permissionResolvers: ({ isAdmin }) => ({
        canDelete: async () => isAdmin,
        canEdit: async () => isAdmin,
    }),
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
})

const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ data, userData }) => {
            return {
                userId: userData.id,
                emailAddress: data.emailAddress,
            }
        },
    },
    trigger: {
        onCreated: ({ prisma, userData }) => {
            Trigger(prisma, userData.languages).createEmail(userData.id)
        },
    },
    yup: { create: emailsCreate },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, emailAddress: true }),
    label: (select) => select.emailAddress,
})

export const EmailModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.email,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    validate: validator(),
})