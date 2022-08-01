import { PrismaType, RecursivePartial } from "../../types";
import { Profile, ProfileEmailUpdateInput, ProfileUpdateInput, Session, Success, Tag, TagCreateInput, TagHidden, User, UserDeleteInput } from "../../schema/types";
import { sendResetPasswordLink, sendVerificationLink } from "../../worker/email/queue";
import { addJoinTablesHelper, addSupplementalFields, FormatConverter, GraphQLInfo, modelToGraphQL, padSelect, PartialGraphQLInfo, readOneHelper, removeJoinTablesHelper, selectHelper, toPartialGraphQLInfo } from "./base";
import { user } from "@prisma/client";
import { CODE, omit, profileUpdateSchema, userTranslationCreate, userTranslationUpdate } from "@local/shared";
import { CustomError } from "../../error";
import bcrypt from 'bcrypt';
import { hasProfanity } from "../../utils/censor";
import { TagModel } from "./tag";
import { EmailModel } from "./email";
import pkg from '@prisma/client';
import { TranslationModel } from "./translation";
import { WalletModel } from "./wallet";
import { genErrorCode } from "../../logger";
import { ResourceListModel } from "./resourceList";
import { TagHiddenModel } from "./tagHidden";
const { AccountStatus } = pkg;

const CODE_TIMEOUT = 2 * 24 * 3600 * 1000;
const HASHING_ROUNDS = 8;
const LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT = 3;
const LOGIN_ATTEMPTS_TO_HARD_LOCKOUT = 10;
const SOFT_LOCKOUT_DURATION = 15 * 60 * 1000;

const tagSelect = {
    id: true,
    created_at: true,
    tag: true,
    stars: true,
    translations: {
        id: true,
        language: true,
        description: true,
    }
}

const joinMapper = { hiddenTags: 'tag', roles: 'role', starredBy: 'user' };
const calculatedFields = ['starredTags', 'hiddenTags'];
export const profileFormatter = (): FormatConverter<User> => ({
    relationshipMap: {
        '__typename': 'Profile',
        'comments': 'Comment',
        'roles': 'Role',
        'emails': 'Email',
        'wallets': 'Wallet',
        'standards': 'Standard',
        'tags': 'Tag',
        'resourceLists': 'ResourceList',
        'organizations': 'Member',
        'projects': 'Project',
        'projectsCreated': 'Project',
        'routines': 'Routine',
        'routinesCreated': 'Routine',
        'starredBy': 'User',
        'starred': 'Star',
        'hiddenTags': 'TagHidden',
        'sentReports': 'Report',
        'reports': 'Report',
        'votes': 'Vote',
    },
    removeCalculatedFields: (partial) => {
        return omit(partial, calculatedFields);
    },
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
})

/**
 * Custom component for email/password validation
 */
export const profileVerifier = () => ({
    /**
     * Generates a URL-safe code for account confirmations and password resets
     * @returns Hashed and salted code, with invalid characters removed
     */
    generateCode(): string {
        return bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '')
    },
    /**
     * Verifies if a confirmation or password reset code is valid
     * @param providedCode Code provided by GraphQL mutation
     * @param storedCode Code stored in user cell in database
     * @param dateRequested Date of request, also stored in database
     * @returns Boolean indicating if the code is valid
     */
    validateCode(providedCode: string | null, storedCode: string | null, dateRequested: Date | null): boolean {
        return Boolean(providedCode) && Boolean(storedCode) && Boolean(dateRequested) &&
            providedCode === storedCode && Date.now() - new Date(dateRequested as Date).getTime() < CODE_TIMEOUT;
    },
    /**
     * Hashes password for safe storage in database
     * @param password Plaintext password
     * @returns Hashed password
     */
    hashPassword(password: string): string {
        return bcrypt.hashSync(password, HASHING_ROUNDS)
    },
    /**
     * Validates a user's password, taking into account the user's account status
     * @param plaintext Plaintext password to check
     * @param user User object
     * @returns Boolean indicating if the password is valid
     */
    validatePassword(plaintext: string, user: any): boolean {
        // A password is only valid if the user is:
        // 1. Not deleted
        // 2. Not locked out
        const status_to_code: any = {
            [AccountStatus.Deleted]: CODE.NoUser,
            [AccountStatus.SoftLocked]: CODE.SoftLockout,
            [AccountStatus.HardLocked]: CODE.HardLockout
        }
        if (user.status in status_to_code)
            throw new CustomError(status_to_code[user.status], 'Account is locked or deleted', { code: genErrorCode('0059'), status: user.status });
        // Validate plaintext password against hash
        return bcrypt.compareSync(plaintext, user.password)
    },
    /**
     * Attemps to log a user in
     * @param password Plaintext password
     * @param user User object
     * @param info Prisma query info
     * @returns Session data
     */
    async logIn(password: string, user: any, prisma: PrismaType): Promise<Session | null> {
        // First, check if the log in fail counter should be reset
        const unable_to_reset = [AccountStatus.HardLocked, AccountStatus.Deleted];
        // If account is not deleted or hard-locked, and lockout duration has passed
        if (!unable_to_reset.includes(user.status) && Date.now() - new Date(user.lastLoginAttempt).getTime() > SOFT_LOCKOUT_DURATION) {
            // Reset log in fail counter
            await prisma.user.update({
                where: { id: user.id },
                data: { logInAttempts: 0 },
            });
        }
        // If account is deleted or hard-locked, throw error
        if (unable_to_reset.includes(user.status))
            throw new CustomError(CODE.BadCredentials, 'Account is locked. Please contact us for assistance', { code: genErrorCode('0060'), status: user.status });
        // If password is valid
        if (this.validatePassword(password, user)) {
            const userData = await prisma.user.update({
                where: { id: user.id },
                data: {
                    logInAttempts: 0,
                    lastLoginAttempt: new Date().toISOString(),
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null
                },
                select: {
                    id: true,
                    theme: true,
                    roles: { select: { role: { select: { title: true } } } }
                }
            });
            return await this.toSession(userData, prisma);
        }
        // If password is invalid
        let new_status: any = AccountStatus.Unlocked;
        let log_in_attempts = user.logInAttempts++;
        if (log_in_attempts > LOGIN_ATTEMPTS_TO_HARD_LOCKOUT) {
            new_status = AccountStatus.HardLocked;
        } else if (log_in_attempts > LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT) {
            new_status = AccountStatus.SoftLocked;
        }
        await prisma.user.update({
            where: { id: user.id },
            data: { status: new_status, logInAttempts: log_in_attempts, lastLoginAttempt: new Date().toISOString() }
        })
        return null;
    },
    /**
     * Updated user object with new password reset code, and sends email to user with reset link
     * @param user User object
     */
    async setupPasswordReset(user: any, prisma: PrismaType): Promise<boolean> {
        // Generate new code
        const resetPasswordCode = this.generateCode();
        // Store code and request time in user row
        const updatedUser = await prisma.user.update({
            where: { id: user.id },
            data: { resetPasswordCode, lastResetPasswordReqestAttempt: new Date().toISOString() },
            select: { emails: { select: { emailAddress: true } } }
        })
        // Send new verification emails
        for (const email of updatedUser.emails) {
            sendResetPasswordLink(email.emailAddress, user.id, resetPasswordCode);
        }
        return true;
    },
    /**
    * Updates email object with new verification code, and sends email to user with link
    * @param user User object
    */
    async setupVerificationCode(emailAddress: string, prisma: PrismaType): Promise<void> {
        // Generate new code
        const verificationCode = this.generateCode();
        // Store code and request time in email row
        const email = await prisma.email.update({
            where: { emailAddress },
            data: { verificationCode, lastVerificationCodeRequestAttempt: new Date().toISOString() },
            select: { userId: true }
        })
        // If email is not associated with a user, throw error
        if (!email.userId)
            throw new CustomError(CODE.ErrorUnknown, 'Email not associated with a user', { code: genErrorCode('0061') });
        // Send new verification email
        sendVerificationLink(emailAddress, email.userId, verificationCode);
        // TODO send email to existing emails from user, warning of new email
    },
    /**
     * Validate verification code and update user's account status
     * @param emailAddress Email address string
     * @param userId ID of user who owns email
     * @param code Verification code
     * @param prisma 
     * @returns True if email was is verified
     */
    async validateVerificationCode(emailAddress: string, userId: string, code: string, prisma: PrismaType): Promise<boolean> {
        // Find email data
        const email: any = await prisma.email.findUnique({
            where: { emailAddress },
            select: {
                id: true,
                userId: true,
                verified: true,
                verificationCode: true,
                lastVerificationCodeRequestAttempt: true
            }
        })
        if (!email)
            throw new CustomError(CODE.EmailNotVerified, 'Email not found', { code: genErrorCode('0062') });
        // Check that userId matches email's userId
        if (email.userId !== userId)
            throw new CustomError(CODE.EmailNotVerified, 'Email does not belong to user', { code: genErrorCode('0063') });
        // If email already verified, remove old verification code
        if (email.verified) {
            await prisma.email.update({
                where: { id: email.id },
                data: { verificationCode: null, lastVerificationCodeRequestAttempt: null }
            })
            return true;
        }
        // Otherwise, validate code
        else {
            // If code is correct and not expired
            if (this.validateCode(code, email.verificationCode, email.lastVerificationCodeRequestAttempt)) {
                await prisma.email.update({
                    where: { id: email.id },
                    data: {
                        verified: true,
                        lastVerifiedTime: new Date().toISOString(),
                        verificationCode: null,
                        lastVerificationCodeRequestAttempt: null
                    }
                })
                return true;
            }
            // If email is not verified, set up new verification code
            else if (!email.verified) {
                await this.setupVerificationCode(emailAddress, prisma);
            }
            return false;
        }
    },
    /**
     * Creates session object from user. 
     * Also updates user's lastSessionVerified
     * @param user User object
     * @param prisma 
     * @returns Session object
     */
    async toSession(user: RecursivePartial<user>, prisma: PrismaType): Promise<Session> {
        if (!user.id)
            throw new CustomError(CODE.ErrorUnknown, 'User ID not found', { code: genErrorCode('0064') });
        // Update user's lastSessionVerified
        await prisma.user.update({
            where: { id: user.id },
            data: { lastSessionVerified: new Date().toISOString() }
        })
        // Return shaped session object
        return {
            id: user.id,
            theme: user.theme ?? 'light',
            isLoggedIn: true,
            languages: (user as any)?.languages ? (user as any).languages.map((language: any) => language.language) : null,
        }
    }
})

/**
 * Custom component for importing/exporting data from Vrooli
 * @param state 
 * @returns 
 */
const porter = (prisma: PrismaType) => ({
    /**
     * Import JSON data to Vrooli. Useful if uploading data created offline, or if
     * you're switching from a competitor to Vrooli. :)
     * @param id 
     */
    async importData(data: string): Promise<Success> {
        throw new CustomError(CODE.NotImplemented);
    },
    /**
     * Export data to JSON. Useful if you want to use Vrooli data on your own,
     * or switch to a competitor :(
     * @param id 
     */
    async exportData(id: string): Promise<string> {
        // Find user
        const user = await prisma.user.findUnique({ where: { id }, select: { numExports: true, lastExport: true } });
        if (!user) throw new CustomError(CODE.ErrorUnknown, 'User not found', { code: genErrorCode('0065') });
        throw new CustomError(CODE.NotImplemented)
    },
})

const profileQuerier = (prisma: PrismaType) => ({
    async findProfile(
        userId: string,
        info: GraphQLInfo,
    ): Promise<RecursivePartial<Profile> | null> {
        const partial = toPartialGraphQLInfo(info, profileFormatter().relationshipMap);
        if (!partial)
            throw new CustomError(CODE.InternalError, 'Could not convert info to partial select.', { code: genErrorCode('0190') });
        // Query profile data and tags
        const profileData = await readOneHelper<any>({
            info,
            input: { id: userId },
            model: ProfileModel,
            prisma,
            userId,
        })
        const { starredTags, hiddenTags } = await this.myTags(userId, partial);
        // Format for GraphQL
        let formatted = modelToGraphQL(profileData, partial) as RecursivePartial<Profile>;
        // Return with supplementalfields added
        const data = (await addSupplementalFields(prisma, userId, [formatted], partial))[0] as RecursivePartial<Profile>;
        return {
            ...data,
            starredTags,
            hiddenTags
        }
    },
    /**
     * Custom search for finding tags you have starred/hidden
     */
    async myTags(
        userId: string,
        partial: PartialGraphQLInfo,
    ): Promise<{ starredTags: Tag[], hiddenTags: TagHidden[] }> {
        let starredTags: Tag[] = [];
        let hiddenTags: any[] = [];
        if (partial.starredTags) {
            // Query starred tags
            const data = (await prisma.star.findMany({
                where: {
                    AND: [
                        { byId: userId },
                        { NOT: { tagId: null } }
                    ]
                },
                select: { tag: padSelect(tagSelect) }
            })).map((star: any) => star.tag)
            // Format for GraphQL
            const formatted: any[] = data.map(d => modelToGraphQL(d, partial.starredTags as PartialGraphQLInfo));
            // Return with supplementalfields added
            starredTags = await (TagModel.format as any).addSupplementalFields(prisma, userId, formatted, partial.starredTags as PartialGraphQLInfo) as Tag[];
        }
        if (partial.hiddenTags) {
            // Query hidden tags
            const data = (await prisma.user_tag_hidden.findMany({
                where: { userId },
                select: {
                    id: true,
                    isBlur: true,
                    tag: padSelect(tagSelect)
                }
            }));
            // Format for GraphQL
            const formatted: any[] = data.map(d => modelToGraphQL(d, partial.hiddenTags as PartialGraphQLInfo));
            // Call addsupplementalFields on tags of hidden data
            const tags = await (TagModel.format as any).addSupplementalFields(prisma, userId, formatted.map(f => f.tag), (partial as any).hiddenTags.tag as PartialGraphQLInfo) as Tag[];
            // const tags = (await addSupplementalFields(prisma, userId, formatted.map(f => f.tag), partial.hiddenTags as PartialGraphQLInfo)) as Tag[];
            hiddenTags = data.map((d: any) => ({
                id: d.id,
                isBlur: d.isBlur,
                tag: tags.find(t => t.id === d.tag.id)
            }))
        }
        return { starredTags, hiddenTags };
    },
})

const profileMutater = (prisma: PrismaType) => ({
    async updateProfile(
        userId: string,
        input: ProfileUpdateInput,
        info: GraphQLInfo,
    ): Promise<RecursivePartial<Profile>> {
        profileUpdateSchema.validateSync(input, { abortEarly: false });
        await WalletModel.verify(prisma).verifyHandle('User', userId, input.handle);
        if (hasProfanity(input.name))
            throw new CustomError(CODE.BannedWord, 'User name contains banned word', { code: genErrorCode('0066') });
        TranslationModel.profanityCheck([input]);
        TranslationModel.validateLineBreaks(input, ['bio'], CODE.LineBreaksBio)
        // Convert info to partial select
        const partial = toPartialGraphQLInfo(info, profileFormatter().relationshipMap);
        if (!partial)
            throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0067') });
        // Handle starred tags
        const tagsToCreate: TagCreateInput[] = [];
        const tagsToConnect: string[] = [];
        if (input.starredTagsConnect) {
            // Check if tags exist
            const tags = await prisma.tag.findMany({
                where: { id: { in: input.starredTagsConnect } },
            })
            const connectTags = tags.map(t => t.id);
            const createTags = input.starredTagsConnect.filter(t => !connectTags.includes(t));
            // Add existing tags to tagsToConnect
            tagsToConnect.push(...connectTags);
            // Add new tags to tagsToCreate
            tagsToCreate.push(...createTags.map((t) => typeof t === 'string' ? ({ tag: t }) : t));
        }
        if (input.starredTagsCreate) {
            // Check if tags exist
            const tags = await prisma.tag.findMany({
                where: { tag: { in: input.starredTagsCreate.map(c => c.tag) } },
            })
            const connectTags = tags.map(t => t.id);
            const createTags = input.starredTagsCreate.filter(t => !connectTags.includes(t.tag));
            // Add existing tags to tagsToConnect
            tagsToConnect.push(...connectTags);
            // Add new tags to tagsToCreate
            tagsToCreate.push(...createTags);
        }
        // Create new tags
        const createTagsData = await Promise.all(tagsToCreate.map(async (tag: TagCreateInput) => await TagModel.mutate(prisma).toDBShape(userId, tag)));
        const createdTags = await prisma.$transaction(
            createTagsData.map((data) => prisma.tag.create({ data })),
        );
        // Combine connect IDs and created IDs
        const tagIds = [...tagsToConnect, ...createdTags.map(t => t.id)];
        // Convert tagIds to star creates
        const starredCreate = tagIds.map(tagId => ({ tagId }));
        // Convert starredTagsDisconnect to star deletes
        const starredDelete = input.starredTagsDisconnect ? await prisma.star.findMany({
            where: {
                AND: [
                    { byId: userId },
                    { tagId: { in: input.starredTagsDisconnect } },
                    { NOT: { tagId: null } }
                ]
            },
            select: { id: true }
        }) : [];
        //Create user data
        let userData: { [x: string]: any } = {
            handle: input.handle,
            name: input.name,
            theme: input.theme,
            hiddenTags: await TagHiddenModel.mutate(prisma).relationshipBuilder(userId, input, false),
            resourceLists: await ResourceListModel.mutate(prisma).relationshipBuilder(userId, input, false),
            starred: {
                create: starredCreate,
                delete: starredDelete,
            },
            translations: TranslationModel.relationshipBuilder(userId, input, { create: userTranslationCreate, update: userTranslationUpdate }, false),
        };
        // Update user
        const profileData = await prisma.user.update({
            where: { id: userId },
            data: userData,
            ...selectHelper(partial)
        });
        const { starredTags, hiddenTags } = await profileQuerier(prisma).myTags(userId, partial);
        // Format for GraphQL
        let formatted = modelToGraphQL(profileData, partial) as RecursivePartial<Profile>;
        // Return with supplementalfields added
        const data = (await addSupplementalFields(prisma, userId, [formatted], partial))[0] as RecursivePartial<Profile>;
        return {
            ...data,
            starredTags,
            hiddenTags
        }
    },
    async updateEmails(
        userId: string,
        input: ProfileEmailUpdateInput,
        info: GraphQLInfo,
    ): Promise<RecursivePartial<Profile>> {
        // Check for correct password
        let user = await prisma.user.findUnique({ where: { id: userId } });
        if (!user)
            throw new CustomError(CODE.InternalError, 'User not found', { code: genErrorCode('0068') });
        if (!profileVerifier().validatePassword(input.currentPassword, user))
            throw new CustomError(CODE.BadCredentials, 'Incorrect password', { code: genErrorCode('0069') });
        // Convert input to partial select
        const partial = toPartialGraphQLInfo(info, profileFormatter().relationshipMap);
        if (!partial)
            throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0070') });
        // Create user data
        let userData: { [x: string]: any } = {
            password: input.newPassword ? profileVerifier().hashPassword(input.newPassword) : undefined,
            emails: await EmailModel.mutate(prisma).relationshipBuilder(userId, input, true),
        };
        // Send verification emails
        if (Array.isArray(input.emailsCreate)) {
            for (const email of input.emailsCreate) {
                await profileVerifier().setupVerificationCode(email.emailAddress, prisma);
            }
        }
        // Update user
        user = await prisma.user.update({
            where: { id: userId },
            data: userData,
            ...selectHelper(partial)
        });
        return modelToGraphQL(user, partial);
    },
    //TODO write this function, and make a similar one for organizations and projects
    // /**
    //  * Anonymizes or deletes objects belonging to a user (yourself)
    //  * @param userId The user ID
    //  * @param input The data to anonymize/delete
    //  */
    // async clearUserData(userId: string, input: UserClearDataInput): Promise<Success> {
    //     // Comments
    //     // Organizations
    //     if (input.organization) {
    //         if (input.organization.anonymizeIds) {

    //         }
    //         if (input.organization.deleteIds) {

    //         }
    //         if (input.organization.anonymizeAll) {

    //         }
    //         if (input.organization.deleteAll) {

    //         }
    //     }
    //     // Projects
    //     // Routines
    //     // Runs (can only be deleted, not anonymized)
    //     // Standards
    //     // Tags (can only be anonymized, not deleted)
    // },
    async deleteProfile(userId: string, input: UserDeleteInput): Promise<Success> {
        // Check for correct password
        let user = await prisma.user.findUnique({ where: { id: userId } });
        if (!user)
            throw new CustomError(CODE.InternalError, 'User not found', { code: genErrorCode('0071') });
        if (!profileVerifier().validatePassword(input.password, user))
            throw new CustomError(CODE.BadCredentials, 'Incorrect password', { code: genErrorCode('0072') });
        // Delete user. User's created objects are deleted separately, with explicit confirmation 
        // given by the user. This is to minimize the chance of deleting objects which other users rely on.
        await prisma.user.delete({
            where: { id: userId }
        })
        return { success: true };
    },
})


export const ProfileModel = ({
    prismaObject: (prisma: PrismaType) => prisma.user,
    format: profileFormatter(),
    mutate: profileMutater,
    port: porter,
    query: profileQuerier,
    verify: profileVerifier(),
})