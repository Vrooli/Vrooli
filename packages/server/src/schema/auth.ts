// GraphQL endpoints for all authentication queries and mutations. These include:
// 1. Wallet login
// 2. Email sign up, log in, verification, and password reset
// 3. Guest login
import { gql } from 'apollo-server-express';
import { CODE, COOKIE, emailLogInSchema, emailSignUpSchema, passwordSchema, emailRequestPasswordChangeSchema, ROLES } from '@local/shared';
import { CustomError } from '../error';
import { generateNonce, randomString, serializedAddressToBech32, verifySignedMessage } from '../auth/walletAuth';
import { generateSessionToken } from '../auth/auth.js';
import { IWrap, RecursivePartial } from '../types';
import { WalletCompleteInput, EmailLogInInput, EmailSignUpInput, EmailRequestPasswordChangeInput, EmailResetPasswordInput, WalletInitInput, Session, Success, WalletComplete } from './types';
import { GraphQLResolveInfo } from 'graphql';
import { Context } from '../context';
import { profileValidater } from '../models';
import { hasProfanity } from '../utils/censor';
import pkg from '@prisma/client';
import { rateLimit } from '../rateLimit';
const { AccountStatus, ResourceListUsedFor } = pkg;

const NONCE_VALID_DURATION = 5 * 60 * 1000; // 5 minutes

export const typeDef = gql`
    enum AccountStatus {
        Deleted
        Unlocked
        SoftLocked
        HardLocked
    }

    input EmailLogInInput {
        email: String
        password: String
        verificationCode: String
    }

    input EmailSignUpInput {
        name: String!
        email: String!
        theme: String!
        marketingEmails: Boolean!
        password: String!
        confirmPassword: String!
    }

    input EmailRequestPasswordChangeInput {
        email: String!
    }

    input EmailResetPasswordInput {
        id: ID!
        code: String!
        newPassword: String!
    }

    input WalletInitInput {
        stakingAddress: String!
        nonceDescription: String
    }

    input WalletCompleteInput {
        stakingAddress: String!
        signedPayload: String!
    }

    type WalletComplete {
        firstLogIn: Boolean!
        session: Session
        wallet: Wallet
    }

    type Session {
        id: ID
        roles: [String!]!
        theme: String!
        languages: [String!]
    }

    extend type Mutation {
        emailLogIn(input: EmailLogInInput!): Session!
        emailSignUp(input: EmailSignUpInput!): Session!
        emailRequestPasswordChange(input: EmailRequestPasswordChangeInput!): Success!
        emailResetPassword(input: EmailResetPasswordInput!): Session!
        guestLogIn: Session!
        logOut: Success!
        validateSession: Session!
        walletInit(input: WalletInitInput!): String!
        walletComplete(input: WalletCompleteInput!): WalletComplete!
    }
`

export const resolvers = {
    AccountStatus: AccountStatus,
    Mutation: {
        emailLogIn: async (_parent: undefined, { input }: IWrap<EmailLogInInput>, context: Context, info: GraphQLResolveInfo): Promise<Session> => {
            await rateLimit({ context, info, max: 100 });
            console.log('email log in start', JSON.stringify(input))
            // Validate arguments with schema
            emailLogInSchema.validateSync(input, { abortEarly: false });
            let user;
            // If email not supplied, check if session is valid
            if (!input.email) {
                console.log('email not supplied, checking session')
                if (!context.req.userId) throw new CustomError(CODE.BadCredentials, 'Must supply email if not logged in');
                // Find user by id
                user = await context.prisma.user.findUnique({
                    where: { id: context.req.userId },
                    select: {
                        id: true,
                        theme: true,
                        roles: { select: { role: { select: { title: true } } } }
                    }
                });
                if (!user) throw new CustomError(CODE.InternalError, 'User not found');
                // Validate verification code
                if (input.verificationCode) {
                    console.log('validating verification code a')
                    if (!input.verificationCode.includes(':')) throw new CustomError(CODE.InvalidArgs, 'Invalid verification code');
                    const [, verificationCode] = input.verificationCode.split(':');
                    // Find all emails for user
                    const emails = await context.prisma.email.findMany({
                        where: {
                            AND: [
                                { userId: user.id },
                                { verificationCode },
                            ]
                        }
                    });
                    console.log('got unverified emails', JSON.stringify(emails))
                    if (emails.length === 0) throw new CustomError(CODE.ErrorUnknown, 'Invalid email or expired verification code');
                    const verified = await profileValidater().validateVerificationCode(emails[0].emailAddress, user.id, verificationCode, context.prisma);
                    if (!verified) throw new CustomError(CODE.BadCredentials, 'Could not verify code.');
                }
                return profileValidater().toSession(user);
            }
            // If email supplied, validate
            else {
                console.log('email supplied, validating')
                const email = await context.prisma.email.findUnique({ where: { emailAddress: input.email ?? '' } });
                if (!email) throw new CustomError(CODE.BadCredentials);
                // Find user
                user = await context.prisma.user.findUnique({ where: { id: email.userId ?? '' } });
                if (!user) throw new CustomError(CODE.InternalError, 'User not found');
                // Check for password in database, if doesn't exist, send a password reset link
                if (!Boolean(user.password)) {
                    await profileValidater().setupPasswordReset(user, context.prisma);
                    throw new CustomError(CODE.MustResetPassword);
                }
                // Validate verification code, if supplied
                if (input.verificationCode) {
                    if (!input.verificationCode.includes(':')) throw new CustomError(CODE.InvalidArgs, 'Invalid verification code');
                    const [, verificationCode] = input.verificationCode.split(':');
                    await profileValidater().validateVerificationCode(email.emailAddress, user.id, verificationCode, context.prisma);
                }
                // Create new session
                const session = await profileValidater().logIn(input?.password as string, user, context.prisma);
                if (session) {
                    // Set session token
                    await generateSessionToken(context.res, session);
                    return session;
                } else {
                    throw new CustomError(CODE.BadCredentials);
                }
            }
        },
        emailSignUp: async (_parent: undefined, { input }: IWrap<EmailSignUpInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            await rateLimit({ context, info, max: 10 });
            // Validate input format
            emailSignUpSchema.validateSync(input, { abortEarly: false });
            // Find user role to give to new user
            const roles = await context.prisma.role.findMany({ select: { id: true, title: true } });
            const actorRoleId = roles.filter((r: any) => r.title === ROLES.Actor)[0].id;
            if (!actorRoleId) throw new CustomError(CODE.ErrorUnknown);
            // Check for censored words
            if (hasProfanity(input.name)) throw new CustomError(CODE.BannedWord);
            // Check if email exists
            const existingEmail = await context.prisma.email.findUnique({ where: { emailAddress: input.email ?? '' } });
            if (existingEmail) throw new CustomError(CODE.EmailInUse);
            // Create user object
            const user = await context.prisma.user.create({
                data: {
                    name: input.name,
                    password: profileValidater().hashPassword(input.password),
                    theme: input.theme,
                    status: AccountStatus.Unlocked,
                    emails: {
                        create: [
                            { emailAddress: input.email },
                        ]
                    },
                    roles: {
                        create: [{ role: { connect: { id: actorRoleId } } }]
                    },
                    resourceLists: {
                        create: [
                            {
                                usedFor: ResourceListUsedFor.Learn,
                            },
                            {
                                usedFor: ResourceListUsedFor.Research,
                            },
                            {
                                usedFor: ResourceListUsedFor.Develop
                            }
                        ]
                    }
                }
            });
            if (!user) throw new CustomError(CODE.ErrorUnknown);
            // Create session from user object
            const session = profileValidater().toSession(user);
            // Set up session token
            await generateSessionToken(context.res, session);
            // Send verification email
            await profileValidater().setupVerificationCode(input.email, context.prisma);
            // Return user data
            return session;
        },
        emailRequestPasswordChange: async (_parent: undefined, { input }: IWrap<EmailRequestPasswordChangeInput>, context: Context, info: GraphQLResolveInfo): Promise<Success> => {
            await rateLimit({ context, info, max: 10 });
            // Validate input format
            emailRequestPasswordChangeSchema.validateSync(input, { abortEarly: false });
            // Validate email address
            const email = await context.prisma.email.findUnique({ where: { emailAddress: input.email ?? '' } });
            if (!email) throw new CustomError(CODE.EmailNotFound);
            // Find user
            let user = await context.prisma.user.findUnique({ where: { id: email.userId ?? '' } });
            if (!user) throw new CustomError(CODE.NoUser);
            // Generate and send password reset code
            const success = await profileValidater().setupPasswordReset(user, context.prisma);
            return { success };
        },
        emailResetPassword: async (_parent: undefined, { input }: IWrap<EmailResetPasswordInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            await rateLimit({ context, info, max: 10 });
            // Validate input format
            passwordSchema.validateSync(input.newPassword, { abortEarly: false });
            // Find user
            let user = await context.prisma.user.findUnique({
                where: { id: input.id },
                select: {
                    id: true,
                    status: true,
                    theme: true,
                    resetPasswordCode: true,
                    lastResetPasswordReqestAttempt: true,
                    emails: { select: { emailAddress: true } },
                    roles: { select: { role: { select: { title: true } } } }
                }
            });
            if (!user) throw new CustomError(CODE.ErrorUnknown);
            // If code is invalid
            if (!profileValidater().validateCode(input.code, user.resetPasswordCode ?? '', user.lastResetPasswordReqestAttempt as Date)) {
                // Generate and send new code
                await profileValidater().setupPasswordReset(user, context.prisma);
                // Return error
                throw new CustomError(CODE.InvalidResetCode);
            }
            // Remove request data from user, and set new password
            await context.prisma.user.update({
                where: { id: user.id as unknown as string },
                data: {
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null,
                    password: profileValidater().hashPassword(input.newPassword)
                }
            })
            // Return session
            return profileValidater().toSession(user);
        },
        guestLogIn: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            await rateLimit({ context, info, max: 500 });
            // Create session
            const session: RecursivePartial<Session> = {
                roles: [ROLES.Guest],
                theme: 'light',
            }
            // Set up session token
            await generateSessionToken(context.res, session);
            return session;
        },
        logOut: async (_parent: undefined, _args: undefined, { res }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            res.clearCookie(COOKIE.Session);
            return { success: true };
        },
        validateSession: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            await rateLimit({ context, info, max: 5000 });
            // If session is expired
            if (!context.req.userId || !Array.isArray(context.req.roles) || context.req.roles.length === 0) {
                context.res.clearCookie(COOKIE.Session);
                throw new CustomError(CODE.SessionExpired);
            }
            // If guest, return default session
            if (context.req.roles.includes(ROLES.Guest)) {
                return {
                    roles: [ROLES.Guest],
                    theme: 'light',
                }
            }
            // Otherwise, check if session can be verified from userId
            const userData = await context.prisma.user.findUnique({
                where: { id: context.req.userId },
                select: {
                    id: true,
                    status: true,
                    theme: true,
                    roles: { select: { role: { select: { title: true } } } }
                }
            });
            if (userData) return profileValidater().toSession(userData);
            // If user data failed to fetch, clear session and return error
            context.res.clearCookie(COOKIE.Session);
            throw new CustomError(CODE.ErrorUnknown);
        },
        /**
         * Starts handshake for establishing trust between backend and user wallet
         * @returns Nonce that wallet must sign and send to walletComplete endpoint
         */
        walletInit: async (_parent: undefined, { input }: IWrap<WalletInitInput>, context: Context, info: GraphQLResolveInfo): Promise<string> => {
            await rateLimit({ context, info, max: 100 });
            // // Make sure that wallet is on mainnet (i.e. starts with 'stake1')
            const deserializedStakingAddress = serializedAddressToBech32(input.stakingAddress);
            if (!deserializedStakingAddress.startsWith('stake1')) throw new CustomError(CODE.InvalidArgs, 'Must use wallet on mainnet');
            // Generate nonce for handshake
            const nonce = await generateNonce(input.nonceDescription as string | undefined);
            // Find existing wallet data in database
            let walletData = await context.prisma.wallet.findUnique({
                where: {
                    stakingAddress: input.stakingAddress,
                },
                select: {
                    id: true,
                    verified: true,
                    userId: true,
                }
            });
            // If wallet exists, update with new nonce
            if (walletData) {
                await context.prisma.wallet.update({
                    where: { id: walletData.id },
                    data: {
                        nonce: nonce,
                        nonceCreationTime: new Date().toISOString(),
                    }
                })
            }
            // If wallet data doesn't exist, create
            if (!walletData) {
                walletData = await context.prisma.wallet.create({
                    data: {
                        stakingAddress: input.stakingAddress,
                        nonce: nonce,
                        nonceCreationTime: new Date().toISOString(),
                    },
                    select: {
                        id: true,
                        verified: true,
                        userId: true,
                    }
                })
            }
            return nonce;
        },
        // Verify that signed message from user wallet has been signed by the correct public address
        walletComplete: async (_parent: undefined, { input }: IWrap<WalletCompleteInput>, context: Context, info: GraphQLResolveInfo): Promise<WalletComplete> => {
            await rateLimit({ context, info, max: 100 });
            // Find wallet with public address
            const walletData = await context.prisma.wallet.findUnique({
                where: { stakingAddress: input.stakingAddress },
                select: {
                    id: true,
                    nonce: true,
                    nonceCreationTime: true,
                    user: {
                        select: { id: true, theme: true, languages: { select: { language: true } } }
                    },
                    verified: true,
                }
            });
            console.log('got wallet data in complete', walletData?.id);
            // If wallet doesn't exist, throw error
            if (!walletData) throw new CustomError(CODE.InvalidArgs);
            // If nonce expired, throw error
            if (!walletData.nonce || !walletData.nonceCreationTime || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION) throw new CustomError(CODE.NonceExpired)
            // Verify that message was signed by wallet address
            const walletVerified = verifySignedMessage(input.stakingAddress, walletData.nonce, input.signedPayload);
            if (!walletVerified) throw new CustomError(CODE.Unauthorized);
            let userData = walletData.user;
            // If user doesn't exist, either create new user, or assign to existing user
            let firstLogIn = false;
            if (!userData?.id) {
                // If signed in, query existing user data
                if (context.req.userId) {
                    userData = await context.prisma.user.findUnique({
                        where: { id: context.req.userId },
                        select: { id: true, theme: true, languages: { select: { language: true } } }
                    })
                }
                // If not signed in, create new user
                else {
                    console.log('wallet complete user DID NOT EXIST')
                    firstLogIn = true;
                    const roles = await context.prisma.role.findMany({ select: { id: true, title: true } });
                    const actorRoleId = roles.filter((r: any) => r.title === ROLES.Actor)[0].id;
                    userData = await context.prisma.user.create({
                        data: {
                            name: `user${randomString(8)}`,
                            roles: {
                                create: [{ role: { connect: { id: actorRoleId } } }]
                            },
                            wallets: {
                                connect: { id: walletData.id }
                            },
                            resourceLists: {
                                create: [
                                    {
                                        usedFor: ResourceListUsedFor.Learn,
                                    },
                                    {
                                        usedFor: ResourceListUsedFor.Research,
                                    },
                                    {
                                        usedFor: ResourceListUsedFor.Develop
                                    }
                                ]
                            }
                        },
                        select: { id: true, theme: true, languages: { select: { language: true } } }
                    });
                }
            }
            // If user exists, make sure it is not verified with a different user
            // You can take a wallet from a different user if it's not verified
            else if (context.req.userId && userData.id !== context.req.userId && walletData.verified) throw new CustomError(CODE.Unauthorized, 'Wallet assigned to a different user');
            // Update wallet and remove nonce data
            const wallet = await context.prisma.wallet.update({
                where: { id: walletData.id },
                data: {
                    verified: true,
                    lastVerifiedTime: new Date().toISOString(),
                    nonce: null,
                    nonceCreationTime: null,
                    userId: userData?.id,
                },
                select: {
                    id: true,
                    name: true,
                    handles: {
                        select: {
                            id: true,
                            handle: true,
                        }
                    },
                    publicAddress: true,
                    stakingAddress: true,
                    verified: true,
                }
            })
            // Create session token
            const session = {
                id: userData?.id,
                languages: userData?.languages?.map((l: any) => l.language) ?? [],
                roles: [ROLES.Actor],
                theme: userData?.theme ?? 'light',
            }
            // Add session token to return payload
            await generateSessionToken(context.res, session);
            return {
                firstLogIn,
                session,
                wallet,
            } as WalletComplete;
        },
    }
}