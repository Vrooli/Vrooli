import { CODE, walletsUpdate, walletUpdate } from "@local/shared";
import { CustomError } from "../../error";
import { PrismaType } from "types";
import { Count, Wallet, WalletUpdateInput } from "../../schema/types";
import { addJoinTablesHelper, CUDInput, CUDResult, FormatConverter, GraphQLModelType, ModelLogic, modelToGraphQL, relationshipToPrisma, RelationshipTypes, removeJoinTablesHelper, selectHelper, ValidateMutationsInput } from "./base";
import { hasProfanity } from "../../utils/censor";
import { genErrorCode } from "../../logger";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { handles: 'handle' };
export const walletFormatter = (): FormatConverter<Wallet> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Wallet,
        'handles': GraphQLModelType.Handle,
        'user': GraphQLModelType.User,
        'organization': GraphQLModelType.Organization,
        'project': GraphQLModelType.Project,
    },
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
})

export const walletVerifier = (prisma: PrismaType) => ({
    /**
     * Maps GraphQLModelType to wallet relationship field
     */
    walletOwnerMap: {
        [GraphQLModelType.User]: 'userId',
        [GraphQLModelType.Organization]: 'organizationId',
        [GraphQLModelType.Project]: 'projectId',
    },
    /**
     * Verify that a handle is owned by a wallet, that is owned by an object. 
     * Throws error on failure. 
     * Allows checks for profanity, cause why not
     * @params for The type of object that the wallet is owned by (i.e. Project, Organization, User)
     * @params forId The ID of the object that the wallet is owned by
     * @params handle The handle to verify
     */
    async verifyHandle(forType: GraphQLModelType.User | GraphQLModelType.Organization | GraphQLModelType.Project, forId: string, handle: string | null | undefined): Promise<void> {
        if (!handle) return;
        // Check that handle belongs to one of user's wallets
        const wallets = await prisma.wallet.findMany({
            where: { [this.walletOwnerMap[forType]]: forId },
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
            throw new CustomError(CODE.InvalidArgs, 'Selected handle cannot be verified.', { code: genErrorCode('0119') });
        // Check for censored words
        if (hasProfanity(handle))
            throw new CustomError(CODE.BannedWord, 'Handle contains banned word', { code: genErrorCode('0120') });
    }
})

export const walletMutater = (prisma: PrismaType) => ({
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'wallets',
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by wallets (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] });
        const { update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: [],
            updateMany: updateMany as { where: { id: string }, data: WalletUpdateInput }[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<unknown, WalletUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId)
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0121') });
        if (createMany) {
            // Not allowed to create wallets this way
            throw new CustomError(CODE.InternalError, 'Not allowed to create wallets with this method', { code: genErrorCode('0122') });
        }
        if (updateMany) {
            walletsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            // Make sure wallets are owned by user or user is an admin/owner of organization
            const wallets = await prisma.wallet.findMany({
                where: {
                    AND: [
                        { id: { in: updateMany.map(wallet => wallet.where.id) } },
                        { userId }, //TODO
                    ],
                },
            });
            if (wallets.length !== updateMany.length)
                throw new CustomError(CODE.NotYourWallet, 'At least one of these wallets is not yours', { code: genErrorCode('0123') });
        }
    },
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<unknown, WalletUpdateInput>): Promise<CUDResult<Wallet>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Not allowed to create wallets this way
            throw new CustomError(CODE.InternalError, 'Not allowed to create wallets this way', { code: genErrorCode('0124') });
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.wallet.findFirst({
                    where: {
                        AND: [
                            input.where,
                            { userId },// TODO or organization
                        ]
                    }
                })
                if (!object)
                    throw new CustomError(CODE.NotFound, "Wallet not found", { code: genErrorCode('0125') });
                // Update
                object = await prisma.wallet.update({
                    where: input.where,
                    data: input.data,
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
            const wallets = await prisma.wallet.findMany({
                where: {
                    AND: [
                        { id: { in: deleteMany } },
                        { userId },//TODO
                    ]
                },
                select: {
                    id: true,
                    verified: true,
                }
            })
            if (!wallets)
                throw new CustomError(CODE.NotFound, "Wallet not found", { code: genErrorCode('0126') });
            // Check if user has at least one verified authentication method, besides the one being deleted
            const numberOfVerifiedWalletDeletes = wallets.filter(wallet => wallet.verified).length;
            const verifiedEmailsCount = await prisma.email.count({
                where: { userId, verified: true }//TODO or organizationId
            })
            const verifiedWalletsCount = await prisma.wallet.count({
                where: { userId, verified: true }//TODO or organizationId
            })
            const wontHaveVerifiedEmail = verifiedEmailsCount <= 0;
            const wontHaveVerifiedWallet = numberOfVerifiedWalletDeletes >= verifiedWalletsCount;
            if (wontHaveVerifiedEmail || wontHaveVerifiedWallet)
                throw new CustomError(CODE.InternalError, "Cannot delete all verified authentication methods", { code: genErrorCode('0127') });
            // Delete
            deleted = await prisma.wallet.deleteMany({
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

export const WalletModel = ({
    prismaObject: (prisma: PrismaType) => prisma.wallet,
    format: walletFormatter(),
    mutate: walletMutater,
    verify: walletVerifier,
})

//==============================================================
/* #endregion Model */
//==============================================================