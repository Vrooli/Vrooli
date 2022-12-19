import { Prisma } from "@prisma/client";
import { walletValidation } from '@shared/validation';
import { permissionsSelectHelper } from "../builders";
import { CustomError } from "../events";
import { Wallet, WalletUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { oneIsPublic } from "../utils";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { Formatter, Validator, Mutater, Displayer, ModelLogic } from "./types";
import { SelectWrap } from "../builders/types";

type Model = {
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
}

const __typename = 'Wallet' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
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
    }
})

/**
* Maps GraphQLModelType to wallet relationship field
*/
const walletOwnerMap = {
    User: 'userId',
    Organization: 'organizationId',
    Project: 'projectId',
}

/**
 * Verify that a handle is owned by a wallet, that is owned by an object. 
 * Throws error on failure. 
 * Allows checks for profanity, cause why not
 * @params for The type of object that the wallet is owned by (i.e. Project, Organization, User)
 * @params forId The ID of the object that the wallet is owned by
 * @params handle The handle to verify
 * @params languages Preferred languages for error messages
 */
export const verifyHandle = async (
    prisma: PrismaType,
    forType: 'User' | 'Organization' | 'Project',
    forId: string,
    handle: string | null | undefined,
    languages: string[]
): Promise<void> => {
    if (!handle) return;
    // Check that handle belongs to one of user's wallets
    const wallets = await prisma.wallet.findMany({
        where: { [walletOwnerMap[forType]]: forId },
        select: {
            handles: {
                select: {
                    handle: true,
                }
            }
        }
    });
    const found = Boolean(wallets.find(w => w.handles.find(h => h.handle === handle)));
    if (!found)
        throw new CustomError('0019', 'ErrorUnknown', languages);
    // Check for censored words
    if (hasProfanity(handle))
        throw new CustomError('0120', 'BannedWord', languages);
}

const validator = (): Validator<Model> => ({
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
})

const mutater = (): Mutater<Model> => ({
    shape: {
        update: async ({ data }) => data,
    },
    yup: walletValidation,
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name ?? '',
})

export const WalletModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.wallet,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,//mutater(),
    validate: validator(),
})