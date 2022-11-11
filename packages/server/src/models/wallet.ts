import { Prisma } from "@prisma/client";
import { CODE } from "@shared/consts";
import { walletsUpdate } from '@shared/validation';
import { CustomError, genErrorCode } from "../events";
import { Wallet, WalletUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { hasProfanity } from "../utils/censor";
import { cudHelper } from "./actions";
import { relationshipBuilderHelper } from "./builder";
import { FormatConverter, CUDInput, CUDResult, GraphQLModelType, Validator } from "./types";

export const walletFormatter = (): FormatConverter<Wallet, any> => ({
    relationshipMap: {
        '__typename': 'Wallet',
        'handles': 'Handle',
        'user': 'User',
        'organization': 'Organization',
    },
})

export const walletValidator = (): Validator<Wallet, Prisma.walletWhereInput> => ({
    /**
     * Maps GraphQLModelType to wallet relationship field
     */
    walletOwnerMap: {
        'User': 'userId',
        'Organization': 'organizationId',
        'Project': 'projectId',
    },
    /**
     * Verify that a handle is owned by a wallet, that is owned by an object. 
     * Throws error on failure. 
     * Allows checks for profanity, cause why not
     * @params for The type of object that the wallet is owned by (i.e. Project, Organization, User)
     * @params forId The ID of the object that the wallet is owned by
     * @params handle The handle to verify
     */
    async verifyHandle(prisma: PrismaType, forType: 'User' | 'Organization' | 'Project', forId: string, handle: string | null | undefined): Promise<void> {
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


    // TODO verify handle for create/update


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

export const walletMutater = (prisma: PrismaType) => ({
    async relationshipBuilder(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'wallets',
    ): Promise<{ [x: string]: any } | undefined> {
        return relationshipBuilderHelper({
            data,
            relationshipName,
            isAdd,
            isTransferable: false,
            userId,
        });
    },
    async cud(params: CUDInput<any, WalletUpdateInput>): Promise<CUDResult<Wallet>> {
        return cudHelper({
            ...params,
            objectType: 'Wallet',
            prisma,
            yup: { yupCreate: walletsUpdate, yupUpdate: walletsUpdate },
            shape: {
                shapeCreate: () => {
                    // Not allowed to create wallets with cud metho
                    throw new CustomError(CODE.InternalError, 'Not allowed to create wallets this way', { code: genErrorCode('0124') });
                },
                shapeUpdate: (_, cuData) => cuData,
            }
        })
    },
})

export const WalletModel = ({
    prismaObject: (prisma: PrismaType) => prisma.wallet,
    format: walletFormatter(),
    mutate: walletMutater,
    type: 'Wallet' as GraphQLModelType,
    validate: walletValidator(),
})