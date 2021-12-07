import { gql } from 'apollo-server-express';
import bcrypt from 'bcrypt';
import { CODE, COOKIE, logInSchema, passwordSchema, signUpSchema, requestPasswordChangeSchema, ROLES } from '@local/shared';
import { CustomError, validateArgs } from '../error';
import { generateToken } from '../auth/auth';
import { sendResetPasswordLink, sendVerificationLink } from '../worker/email/queue';
import { PrismaSelect } from '@paljs/plugins';
import { userFromEmail, upsertUser } from '../db/models/user';
import pkg from '@prisma/client';
const { AccountStatus } = pkg;

const HASHING_ROUNDS = 8;
const LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT = 3;
const SOFT_LOCKOUT_DURATION = 15 * 60 * 1000;
const REQUEST_PASSWORD_RESET_DURATION = 2 * 24 * 3600 * 1000;
const LOGIN_ATTEMPTS_TO_HARD_LOCKOUT = 10;

export const typeDef = gql`
    enum AccountStatus {
        DELETED
        UNLOCKED
        SOFT_LOCKED
        HARD_LOCKED
    }

    input UserInput {
        id: ID
        username: String
        pronouns: String
        emails: [EmailInput!]
        theme: String
        status: AccountStatus
    }

    type User {
        id: ID!
        username: String
        pronouns: String!
        emails: [Email!]!
        theme: String!
        emailVerified: Boolean!
        status: AccountStatus!
        roles: [UserRole!]!
    }

    extend type Query {
        profile: User!
    }

    extend type Mutation {
        login(
            email: String
            password: String,
            verificationCode: String
        ): User!
        logout: Boolean
        signUp(
            username: String!
            pronouns: String
            email: String!
            theme: String!
            marketingEmails: Boolean!
            password: String!
        ): User!
        updateUser(
            input: UserInput!
            currentPassword: String!
            newPassword: String
        ): User!
        deleteUser(
            id: ID!
            password: String
        ): Boolean
        requestPasswordChange(
            email: String!
        ): Boolean
        resetPassword(
            id: ID!
            code: String!
            newPassword: String!
        ): User!
    }
`

export const resolvers = {
    AccountStatus: AccountStatus,
    Query: {
        profile: async (_parent: undefined, _args: any, context: any, info: any) => {
            // Can only query your own profile
            const userId = context.req.userId;
            if (userId === null || userId === undefined) return new CustomError(CODE.Unauthorized);
            return await context.prisma.user.findUnique({ where: { id: userId }, ...(new PrismaSelect(info).value) });
        }
    },
    Mutation: {
        login: async (_parent: undefined, args: any, context: any, info: any) => {
            const prismaInfo = new PrismaSelect(info).value
            // If username and password wasn't passed, then use the session cookie data to validate
            if (args.username === undefined && args.password === undefined) {
                if (context.req.roles && context.req.roles.length > 0) {
                    let userData = await context.prisma.user.findUnique({ where: { id: context.req.userId }, ...prismaInfo });
                    if (userData) {
                        return userData;
                    }
                    context.res.clearCookie(COOKIE.Session);
                }
                return new CustomError(CODE.BadCredentials);
            }
            // Validate input format
            const validateError = await validateArgs(logInSchema, args);
            if (validateError) return validateError;
            // Get user
            let user = await userFromEmail(args.email, context.prisma);
            // Check for password in database, if doesn't exist, send a password reset link
            if (!user.password) {
                // Generate new code
                const requestCode = bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '');
                // Store code and request time in user row
                await context.prisma.user.update({
                    where: { id: user.id },
                    data: { resetPasswordCode: requestCode, lastResetPasswordReqestAttempt: new Date().toISOString() }
                })
                // Send new verification email
                sendResetPasswordLink(args.email, user.id, requestCode);
                return new CustomError(CODE.MustResetPassword);
            }
            // Validate verification code, if supplied
            if (args.verificationCode === user.id && user.emailVerified === false) {
                user = await context.prisma.user.update({
                    where: { id: user.id },
                    data: { status: AccountStatus.UNLOCKED, emailVerified: true }
                })
            }
            // Reset login attempts after 15 minutes
            const unable_to_reset = [AccountStatus.HARD_LOCKED, AccountStatus.DELETED];
            if (!unable_to_reset.includes(user.status) && Date.now() - new Date(user.lastLoginAttempt).getTime() > SOFT_LOCKOUT_DURATION) {
                user = await context.prisma.user.update({
                    where: { id: user.id },
                    data: { loginAttempts: 0 }
                })
            }
            // Before validating password, let's check to make sure the account is unlocked
            const status_to_code: any = {
                [AccountStatus.DELETED]: CODE.NoUser,
                [AccountStatus.SOFT_LOCKED]: CODE.SoftLockout,
                [AccountStatus.HARD_LOCKED]: CODE.HardLockout
            }
            if (user.status in status_to_code) return new CustomError(status_to_code[user.status]);
            // Now we can validate the password
            const validPassword = bcrypt.compareSync(args.password, user.password);
            if (validPassword) {
                await generateToken(context.res, user.id);
                await context.prisma.user.update({
                    where: { id: user.id },
                    data: { 
                        loginAttempts: 0, 
                        lastLoginAttempt: new Date().toISOString(), 
                        resetPasswordCode: null, 
                        lastResetPasswordReqestAttempt: null 
                    },
                    ...prismaInfo
                })
                // Return user data
                return await context.prisma.user.findUnique({ where: { id: user.id }, ...prismaInfo });
            } else {
                let new_status: any = AccountStatus.UNLOCKED;
                let login_attempts = user.loginAttempts + 1;
                if (login_attempts >= LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT) {
                    new_status = AccountStatus.SOFT_LOCKED;
                } else if (login_attempts > LOGIN_ATTEMPTS_TO_HARD_LOCKOUT) {
                    new_status = AccountStatus.HARD_LOCKED;
                }
                await context.prisma.user.update({
                    where: { id: user.id },
                    data: { status: new_status, loginAttempts: login_attempts, lastLoginAttempt: new Date().toISOString() }
                })
                return new CustomError(CODE.BadCredentials);
            }
        },
        logout: async (_parent: undefined, _args: any, context: any, _info: any) => {
            context.res.clearCookie(COOKIE.Session);
        },
        signUp: async (_parent: undefined, args: any, context: any, info: any) => {
            const prismaInfo = new PrismaSelect(info).value
            // Validate input format
            const validateError = await validateArgs(signUpSchema, args);
            if (validateError) return validateError;
            // Find user role to give to new user
            const actorRole = await context.prisma.role.findUnique({ where: { title: ROLES.Actor } });
            if (!actorRole) return new CustomError(CODE.ErrorUnknown);
            const user = await upsertUser({
                prisma: context.prisma,
                info,
                data: {
                    username: args.username,
                    password: bcrypt.hashSync(args.password, HASHING_ROUNDS),
                    theme: args.theme,
                    status: AccountStatus.UNLOCKED,
                    emails: [{ emailAddress: args.email }],
                    roles: [actorRole]
                }
            })
            await generateToken(context.res, user.id);
            // Send verification email
            sendVerificationLink(args.email, user.id);
            // Return user data
            return await context.prisma.user.findUnique({ where: { id: user.id }, ...prismaInfo });
        },
        updateUser: async (_parent: undefined, args: any, context: any, info: any) => {
            // Must be updating your own
            if(context.req.userId !== args.input.id) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await context.prisma.user.findUnique({ 
                where: { id: args.input.id },
                select: {
                    id: true,
                    password: true,
                }
            });
            if(!bcrypt.compareSync(args.currentPassword, user.password)) return new CustomError(CODE.BadCredentials);
            const updatedUser = await upsertUser({
                prisma: context.prisma,
                info,
                data: args.input
            })
            return updatedUser;
        },
        deleteUser: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must be deleting your own
            if(context.req.userId !== args.input.id) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await context.prisma.user.findUnique({ 
                where: { id: args.id },
                select: {
                    id: true,
                    password: true
                }
            });
            if (!user) return new CustomError(CODE.ErrorUnknown);
            // Make sure correct password is entered
            if(!bcrypt.compareSync(args.password, user.password)) return new CustomError(CODE.BadCredentials);
            // Delete account
            await context.prisma.user.delete({ where: { id: user.id } });
            return true;
        },
        requestPasswordChange: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Validate input format
            const validateError = await validateArgs(requestPasswordChangeSchema, args);
            if (validateError) return validateError;
            // Find user in database
            const user = await userFromEmail(args.email, context.prisma);
            // Generate request code
            const requestCode = bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '');
            // Store code and request time in user row
            await context.prisma.user.update({
                where: { id: user.id },
                data: { resetPasswordCode: requestCode, lastResetPasswordReqestAttempt: new Date().toISOString() }
            })
            // Send email with correct reset link
            sendResetPasswordLink(args.email, user.id, requestCode);
            return true;
        },
        resetPassword: async(_parent: undefined, args: any, context: any, info: any) => {
            // Validate input format
            const validateError = await validateArgs(passwordSchema, args.newPassword);
            if (validateError) return validateError;
            // Find user in database
            const user = await context.prisma.user.findUnique({ 
                where: { id: args.id },
                select: {
                    id: true,
                    resetPasswordCode: true,
                    lastResetPasswordReqestAttempt: true,
                    emails: { select: { emailAddress: true } }
                }
            });
            if (!user) return new CustomError(CODE.ErrorUnknown);
            // Verify request code and that request was made within 48 hours
            if (!user.resetPasswordCode ||
                user.resetPasswordCode !== args.code ||
                Date.now() - new Date(user.lastResetPasswordReqestAttempt).getTime() > REQUEST_PASSWORD_RESET_DURATION) {
                // Generate new code
                const requestCode = bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '');
                // Store code and request time in user row
                await context.prisma.user.update({
                    where: { id: user.id },
                    data: { resetPasswordCode: requestCode, lastResetPasswordReqestAttempt: new Date().toISOString() }
                })
                // Send new verification email
                for (const email of user.emails) {
                    sendResetPasswordLink(email.emailAddress, user.id, requestCode);
                }
                // Return error
                return new CustomError(CODE.InvalidResetCode);
            } 
            // Remove request data from user, and set new password
            await context.prisma.user.update({
                where: { id: user.id },
                data: { 
                    resetPasswordCode: null, 
                    lastResetPasswordReqestAttempt: null,
                    password: bcrypt.hashSync(args.newPassword, HASHING_ROUNDS)
                }
            })
            // Return user data
            const prismaInfo = new PrismaSelect(info).value
            return await context.prisma.user.findUnique({ where: { id: user.id }, ...prismaInfo });
        },
    }
}