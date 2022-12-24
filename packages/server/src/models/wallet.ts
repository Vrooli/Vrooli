import { Prisma } from "@prisma/client";
import { walletValidation } from '@shared/validation';
import { permissionsSelectHelper } from "../builders";
import { CustomError } from "../events";
import { Wallet, WalletUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { oneIsPublic } from "../utils";
import { OrganizationModel } from "./organization";
import { ModelLogic } from "./types";
import { SelectWrap } from "../builders/types";

const __typename = 'Wallet' as const;
const suppFields = [] as const;
export const WalletModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: WalletUpdateInput,
    GqlModel: Wallet,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.walletUpsertArgs['create'],
    PrismaUpdate: Prisma.walletUpsertArgs['update'],
    PrismaModel: Prisma.walletGetPayload<SelectWrap<Prisma.walletSelect>>,
    PrismaSelect: Prisma.walletSelect,
    PrismaWhere: Prisma.walletWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.wallet,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name ?? '',
    },
    format: {
        gqlRelMap: {
            __typename,
            handles: 'Handle',
            user: 'User',
            organization: 'Organization',
        },
        prismaRelMap: {
            __typename,
            handles: 'Handle',
            user: 'User',
            organization: 'Organization',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            update: async ({ data }) => data,
        },
        yup: walletValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: {
            User: {
                private: 5,
                public: 0,
            },
            Organization: {
                private: {
                    noPremium: 1,
                    premium: 5,
                },
                public: 0,
            },
        },
        permissionsSelect: (...params) => ({
            id: true,
            ...permissionsSelectHelper([
                ['organization', 'Organization'],
                ['user', 'User'],
            ], ...params)
        }),
        permissionResolvers: () => ({}),
        owner: (data) => ({
            Organization: data.organization,
            User: data.user,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.walletSelect>(data, [
            ['organization', 'Organization'],
            ['user', 'User'],
        ], languages),
        validations: {
            delete: async ({ deleteMany, prisma, userData }) => {
                // Prevent deleting wallets if it will leave you with less than one verified authentication method
                const allWallets = await prisma.wallet.findMany({
                    where: { user: { id: userData.id } },
                    select: { id: true, verified: true }
                });
                const remainingVerifiedWalletsCount = allWallets.filter(x => !deleteMany.includes(x.id) && x.verified).length;
                const verifiedEmailsCount = await prisma.email.count({
                    where: { user: { id: userData.id }, verified: true },
                });
                if (remainingVerifiedWalletsCount + verifiedEmailsCount < 1)
                    throw new CustomError('0049', 'MustLeaveVerificationMethod', userData.languages);
            }
        },
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }),
        },
    },
})