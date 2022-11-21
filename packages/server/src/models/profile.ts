import { PrismaType, RecursivePartial } from "../types";
import { Profile, ProfileEmailUpdateInput, ProfileUpdateInput, Session, SessionUser, Success, TagCreateInput, UserDeleteInput } from "../schema/types";
import { addJoinTablesHelper, addSupplementalFields, modelToGraphQL, padSelect, removeJoinTablesHelper, selectHelper, toPartialGraphQLInfo } from "./builder";
import { profileUpdateSchema, userTranslationCreate, userTranslationUpdate } from "@shared/validation";
import bcrypt from 'bcrypt';
import { hasProfanity } from "../utils/censor";
import { TagModel } from "./tag";
import { EmailModel } from "./email";
import { TranslationModel } from "./translation";
import { verifyHandle } from "./wallet";
import { Request } from "express";
import { CustomError } from "../events";
import { readOneHelper } from "./actions";
import { FormatConverter, GraphQLInfo, PartialGraphQLInfo, GraphQLModelType } from "./types";
import pkg, { Prisma } from '@prisma/client';
import { sendResetPasswordLink, sendVerificationLink } from "../notify";
import { lineBreaksCheck, profanityCheck } from "./validators";
import { assertRequestFrom } from "../auth/auth";
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
type SupplementalFields = 'starredTags' | 'hiddenTags';
export const profileFormatter = (): FormatConverter<Profile, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'Profile',
        comments: 'Comment',
        roles: 'Role',
        emails: 'Email',
        wallets: 'Wallet',
        resourceLists: 'ResourceList',
        projects: 'Project',
        projectsCreated: 'Project',
        routines: 'Routine',
        routinesCreated: 'Routine',
        starredBy: 'User',
        stars: 'Star',
        starredTags: 'Tag',
        hiddenTags: 'TagHidden',
        sentReports: 'Report',
        reports: 'Report',
        votes: 'Vote',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    supplemental: {
        graphqlFields: ['starredTags', 'hiddenTags'],
        toGraphQL: ({ ids, objects, partial, prisma, userData }) => [
            ['starredTags', async () => {
                if (!userData) return new Array(objects.length).fill([]);
                // Query starred tags
                let data = (await prisma.star.findMany({
                    where: {
                        AND: [
                            { byId: userData.id },
                            { NOT: { tagId: null } }
                        ]
                    },
                    select: { tag: padSelect(tagSelect) }
                })).map((star: any) => star.tag)
                // Format to GraphQL
                data = data.map(r => modelToGraphQL(r, partial.starredTags as PartialGraphQLInfo));
                // Add supplemental fields
                data = await addSupplementalFields(prisma, userData, data, partial.starredTags as PartialGraphQLInfo);
                // Split by id
                const result = ids.map((id) => data.filter(r => r.routineId === id));
                return result;
            }],
            ['hiddenTags', async () => {
                if (!userData) return new Array(objects.length).fill([]);
                // // Query hidden tags
                // let data = (await prisma.user_tag_hidden.findMany({
                //     where: { userId: userData.id },
                //     select: {
                //         id: true,
                //         isBlur: true,
                //         tag: padSelect(tagSelect)
                //     }
                // }));
                // // Format to GraphQL
                // data = data.map(r => modelToGraphQL(r, partial.starredTags as PartialGraphQLInfo));
                // // Add supplemental fields
                // data = await addSupplementalFields(prisma, userData, data, partial.starredTags as PartialGraphQLInfo);
                // return ids.map((d: any) => ({
                //     id: d.id,
                //     isBlur: d.isBlur,
                //     tag: data.find(t => t.id === d.tag.id)
                // }))
                return new Array(objects.length).fill([]);
            }],
        ],
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
     * @param languages Preferred languages to display error messages in
     * @returns Boolean indicating if the password is valid
     */
    validatePassword(plaintext: string, user: any, languages: string[]): boolean {
        // A password is only valid if the user is:
        // 1. Not deleted
        // 2. Not locked out
        // If account is deleted or locked, throw error
        if (user.status === AccountStatus.HardLocked)
            throw new CustomError('0060', 'HardLockout', languages);
        if (user.status === AccountStatus.SoftLocked)
            throw new CustomError('0330', 'SoftLockout', languages);
        if (user.status === AccountStatus.Deleted)
            throw new CustomError('0061', 'AccountDeleted', languages);
        // Validate plaintext password against hash
        return bcrypt.compareSync(plaintext, user.password)
    },
    /**
     * Attemps to log a user in
     * @param password Plaintext password
     * @param user User object
     * @param prisma Prisma type
     * @param req Express request object
     * @returns Session data
     */
    async logIn(password: string, user: any, prisma: PrismaType, req: Request): Promise<Session | null> {
        // First, check if the log in fail counter should be reset
        // If account is NOT deleted or hard-locked, and lockout duration has passed
        if (![AccountStatus.HardLocked, AccountStatus.Deleted].includes(user.status) && Date.now() - new Date(user.lastLoginAttempt).getTime() > SOFT_LOCKOUT_DURATION) {
            // Reset log in fail counter
            await prisma.user.update({
                where: { id: user.id },
                data: { logInAttempts: 0 },
            });
        }
        // If account is deleted or locked, throw error
        if (user.status === AccountStatus.HardLocked)
            throw new CustomError('0060', 'HardLockout', req.languages);
        if (user.status === AccountStatus.SoftLocked)
            throw new CustomError('0330', 'SoftLockout', req.languages);
        if (user.status === AccountStatus.Deleted)
            throw new CustomError('0061', 'AccountDeleted', req.languages);
        // If password is valid
        if (this.validatePassword(password, user, req.languages)) {
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
                    name: true,
                    theme: true,
                    roles: { select: { role: { select: { title: true } } } },
                }
            });
            return await this.toSession(userData, prisma, req);
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
    async setupPasswordReset(user: { id: string, resetPasswordCode: string | null }, prisma: PrismaType): Promise<boolean> {
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
    */
    async setupVerificationCode(emailAddress: string, prisma: PrismaType, languages: string): Promise<void> {
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
            throw new CustomError('0061', 'EmailNotYours', languages);
        // Send new verification email
        sendVerificationLink(emailAddress, email.userId, verificationCode);
        // TODO send email to existing emails from user, warning of new email
    },
    /**
     * Validate verification code and update user's account status
     * @param emailAddress Email address string
     * @param userId ID of user who owns email
     * @param code Verification code
     * @param prisma The Prisma client
     * @param languages Preferred languages to display error messages in
     * @returns True if email was is verified
     */
    async validateVerificationCode(emailAddress: string, userId: string, code: string, prisma: PrismaType, languages: string[]): Promise<boolean> {
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
            throw new CustomError('0062', 'EmailNotFound', languages);
        // Check that userId matches email's userId
        if (email.userId !== userId)
            throw new CustomError('0063', 'EmailNotYours', languages);
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
                await this.setupVerificationCode(emailAddress, prisma, languages);
            }
            return false;
        }
    },
    /**
     * Creates SessionUser object from user.
     * Also updates user's lastSessionVerified time
     * @param user User object
     * @param prisma Prisma type
     * @param req Express request object
     */
    async toSessionUser(user: { id: string }, prisma: PrismaType, req: Partial<Request>): Promise<SessionUser> {
        if (!user.id)
            throw new CustomError('0064', 'NotFound', req.languages ?? ['en']);
        // Update user's lastSessionVerified
        const userData = await prisma.user.update({
            where: { id: user.id },
            data: { lastSessionVerified: new Date().toISOString() },
            select: {
                id: true,
                handle: true,
                languages: { select: { language: true } },
                name: true,
                theme: true,
                premium: { select: { id: true, expiresAt: true } },
            }
        })
        // Calculate langugages, by combining user's languages with languages 
        // in request. Make sure to remove duplicates
        let languages: string[] =  userData.languages.map((l) => l.language).filter(Boolean) as string[];
        if (req.languages) languages.push(...req.languages);
        languages = [...new Set(languages)];
        // Return shaped SessionUser object
        return {
            handle: userData.handle ?? undefined,
            hasPremium: new Date(userData.premium?.expiresAt ?? 0) > new Date(),
            id: user.id,
            languages,
            name: userData.name,
            theme: userData.theme,
        }
    },
    /**
     * Creates session object from user and existing session data
     * @param user User object
     * @param prisma 
     * @param req Express request object (with current session data)
     * @returns Updated session object, with user data added to the START of the users array
     */
    async toSession(user: { id: string }, prisma: PrismaType, req: Partial<Request>): Promise<Session> {
        const sessionUser = await this.toSessionUser(user, prisma, req);
        return {
            __typename: 'Session',
            isLoggedIn: true,
            // Make sure users are unique by id
            users: [sessionUser, ...(req.users ?? []).filter((u: SessionUser) => u.id !== sessionUser.id)],
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
        throw new CustomError('0323', 'NotImplemented', languages);
    },
    /**
     * Export data to JSON. Useful if you want to use Vrooli data on your own,
     * or switch to a competitor :(
     * @param id 
     */
    async exportData(id: string): Promise<string> {
        // Find user
        const user = await prisma.user.findUnique({ where: { id }, select: { numExports: true, lastExport: true } });
        if (!user) throw new CustomError('0065', 'NoUser', languages);
        throw new CustomError('0324', 'NotImplemented', languages)
    },
})

const profileQuerier = (prisma: PrismaType) => ({
    async findProfile(
        req: Request,
        info: GraphQLInfo,
    ): Promise<RecursivePartial<Profile> | null> {
        const userData = assertRequestFrom(req, { isUser: true });
        const partial = toPartialGraphQLInfo(info, profileFormatter().relationshipMap);
        if (!partial)
            throw new CustomError('0190', 'InternalError', req.languages);
        // Query profile data and tags
        const profileData = await readOneHelper<any>({
            info,
            input: { id: userData.id },
            model: ProfileModel,
            prisma,
            req,
        })
        // Format for GraphQL
        let formatted = modelToGraphQL(profileData, partial) as RecursivePartial<Profile>;
        // Return with supplementalfields added
        const data = (await addSupplementalFields(prisma, userData, [formatted], partial))[0] as RecursivePartial<Profile>;
        return data;
    },
})

const profileMutater = (prisma: PrismaType) => ({
    async updateProfile(
        userData: SessionUser,
        input: ProfileUpdateInput,
        info: GraphQLInfo,
    ): Promise<RecursivePartial<Profile>> {
        profileUpdateSchema.validateSync(input, { abortEarly: false });
        await verifyHandle(prisma, 'User', userData.id, input.handle);
        if (hasProfanity(input.name))
            throw new CustomError('0066', 'BannedWord', languages);
        profanityCheck([input], { __typename: 'User' });
        lineBreaksCheck(input, ['bio'], 'LineBreaksBio')
        // Convert info to partial select
        const partial = toPartialGraphQLInfo(info, profileFormatter().relationshipMap);
        if (!partial)
            throw new CustomError('0067', 'InternalError', languages);
        // Handle starred tags
        const tagsToCreate: TagCreateInput[] = [
            ...(input.starredTagsCreate ?? []),
            ...((input.starredTagsConnect ?? []).map(t => ({
                tag: t
            }))),
        ];
        // Create new tags
        const createTagsData = await Promise.all(tagsToCreate.map(async (tag: TagCreateInput) => await TagModel.mutate(prisma).shapeCreate(userData.id, tag)));
        const createdTags = await prisma.$transaction(
            createTagsData.map((data) =>
                prisma.tag.upsert({
                    where: { tag: data.tag },
                    create: data,
                    update: { tag: data.tag }
                })
            ),
        );
        // Combine connect IDs and created IDs
        const tagIds = createdTags.map(t => t.id);
        // Convert tagIds to star creates
        const starredCreate = tagIds.map(tagId => ({ tagId }));
        // Convert starredTagsDisconnect to star deletes
        const starredDelete = input.starredTagsDisconnect ? await prisma.star.findMany({
            where: {
                AND: [
                    { by: { id: userData.id } },
                    { tag: { tag: { in: input.starredTagsDisconnect } } },
                    { NOT: { tagId: null } },
                ]
            },
            select: { id: true }
        }) : [];
        // Create user data
        let uData: Prisma.userUpsertArgs['update'] = {
            handle: input.handle,
            name: input.name ?? undefined,
            theme: input.theme ?? undefined,
            // hiddenTags: await TagHiddenModel.mutate(prisma).relationshipBuilder!(userData.id, input, false),
            starred: {
                create: starredCreate,
                delete: starredDelete,
            },
            translations: TranslationModel.relationshipBuilder(userData.id, input, { create: userTranslationCreate, update: userTranslationUpdate }, false),
        };
        // Update user
        const profileData = await prisma.user.update({
            where: { id: userData.id },
            data: uData,
            ...selectHelper(partial)
        });
        // Format for GraphQL
        let formatted = modelToGraphQL(profileData, partial) as RecursivePartial<Profile>;
        // Return with supplementalfields added
        const data = (await addSupplementalFields(prisma, userData, [formatted], partial))[0] as RecursivePartial<Profile>;
        return data
    },
    async updateEmails(
        userId: string,
        input: ProfileEmailUpdateInput,
        info: GraphQLInfo,
    ): Promise<RecursivePartial<Profile>> {
        // Check for correct password
        let user = await prisma.user.findUnique({ where: { id: userId } });
        if (!user)
            throw new CustomError('0068', 'NoUser', languages);
        if (!profileVerifier().validatePassword(input.currentPassword, user))
            throw new CustomError('0069', 'BadCredentials', languages);
        // Convert input to partial select
        const partial = toPartialGraphQLInfo(info, profileFormatter().relationshipMap);
        if (!partial)
            throw new CustomError('0070', 'InternalError', languages);
        // Create user data
        let userData: { [x: string]: any } = {
            password: input.newPassword ? profileVerifier().hashPassword(input.newPassword) : undefined,
            emails: await EmailModel.mutate(prisma).relationshipBuilder!(userId, input, true),
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
            throw new CustomError('0071', 'NoUser', languages);
        if (!profileVerifier().validatePassword(input.password, user))
            throw new CustomError('0072', 'BadCredentials', languages);
        // Delete user. User's created objects are deleted separately, with explicit confirmation 
        // given by the user. This is to minimize the chance of deleting objects which other users rely on. TODO
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
    type: 'Profile' as GraphQLModelType,
    verify: profileVerifier(),
})