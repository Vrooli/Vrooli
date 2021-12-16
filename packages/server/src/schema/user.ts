import { gql } from 'apollo-server-express';
import { CODE, COOKIE, logInSchema, passwordSchema, signUpSchema, requestPasswordChangeSchema, ROLES } from '@local/shared';
import { CustomError, validateArgs } from '../error';
import { generateToken } from '../auth/auth';
import { UserModel } from '../models/user';
import pkg from '@prisma/client';
const { AccountStatus } = pkg;

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
            password: String
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
        reportUser(id: ID!): Boolean!
    }
`

export const resolvers = {
    AccountStatus: AccountStatus,
    Query: {
        profile: async (_parent: undefined, _args: any, context: any, info: any) => {
            return await new UserModel(context.prisma).findById(context.req.userId, info);
        }
    },
    Mutation: {
        login: async (_parent: undefined, args: any, context: any, info: any) => {
            const userModel = new UserModel(context.prisma);
            // If username and password are not provided, attempt to log in using session cookie
            if (!Boolean(args.username) && !Boolean(args.password)) {
                // If session is expired
                if (!Array.isArray(context.req.roles) || context.req.roles.length === 0) {
                    context.res.clearCookie(COOKIE.Session);
                    return new CustomError(CODE.SessionExpired);
                }
                let userData = await userModel.findById(context.req.userId, info);
                if (userData) return userData;
                // If user data failed to fetch, clear session and return error
                context.res.clearCookie(COOKIE.Session);
                return new CustomError(CODE.ErrorUnknown);
            }
            // Validate arguments with schema
            const validateError = await validateArgs(logInSchema, args);
            if (validateError) return validateError;
            // Get user
            let user = await new UserModel(context.prisma).fromEmail(args.email);
            // Check for password in database, if doesn't exist, send a password reset link
            if (!Boolean(user.password)) {
                await new UserModel(context.prisma).setupPasswordReset(user);
                return new CustomError(CODE.MustResetPassword);
            }
            // Validate verification code, if supplied
            if (Boolean(args.verificationCode)) {
                user = await new UserModel(context.prisma).validateVerificationCode(user, args.verificationCode);
            }
            user = await userModel.logIn(args.password, user, info);
            if (Boolean(user)) {
                // Set session token
                await generateToken(context.res, user.id);
                return user;
            } else {
                return new CustomError(CODE.BadCredentials);
            }
        },
        logout: async (_parent: undefined, _args: any, context: any, _info: any) => {
            context.res.clearCookie(COOKIE.Session);
        },
        signUp: async (_parent: undefined, args: any, context: any, info: any) => {
            // Validate input format
            const validateError = await validateArgs(signUpSchema, args);
            if (validateError) return validateError;
            // Find user role to give to new user
            const actorRole = await context.prisma.role.findUnique({ where: { title: ROLES.Actor } });
            if (!actorRole) return new CustomError(CODE.ErrorUnknown);
            // Create user object
            const user = await new UserModel(context.prisma).upsertUser({
                username: args.username,
                password: UserModel.hashPassword(args.password),
                theme: args.theme,
                status: AccountStatus.UNLOCKED,
                emails: [{ emailAddress: args.email }],
                roles: [actorRole]
            }, info);
            // Set up session token
            await generateToken(context.res, user.id);
            // Send verification email
            await new UserModel(context.prisma).setupVerificationCode(user);
            // Return user data
            return await new UserModel(context.prisma).findById(user.id, info)
        },
        updateUser: async (_parent: undefined, args: any, context: any, info: any) => {
            // Must be updating your own
            if (context.req.userId !== args.input.id) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await context.prisma.user.findUnique({ where: { id: args.input.id } });
            if (!user) return new CustomError(CODE.InvalidArgs);
            if (!UserModel.validatePassword(args.currentPassword, user)) return new CustomError(CODE.BadCredentials);
            // Update user
            return await new UserModel(context.prisma).upsertUser(args.input, info);
        },
        deleteUser: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must be deleting your own
            if (context.req.userId !== args.input.id) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await context.prisma.user.findUnique({ where: { id: args.id } });
            if (!user) return new CustomError(CODE.InvalidArgs);
            if (!UserModel.validatePassword(args.password, user)) return new CustomError(CODE.BadCredentials);
            // Delete user
            await new UserModel(context.prisma).delete(user.id);
            return true;
        },
        requestPasswordChange: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Validate input format
            const validateError = await validateArgs(requestPasswordChangeSchema, args);
            if (validateError) return validateError;
            // Find user in database
            const user = await new UserModel(context.prisma).fromEmail(args.email);
            // Generate and send password reset code
            await new UserModel(context.prisma).setupPasswordReset(user);
            return true;
        },
        resetPassword: async (_parent: undefined, args: any, context: any, info: any) => {
            // Validate input format
            const validateError = await validateArgs(passwordSchema, args.newPassword);
            if (validateError) return validateError;
            // Find user in database
            let user = await context.prisma.user.findUnique({
                where: { id: args.id },
                select: {
                    id: true,
                    status: true,
                    resetPasswordCode: true,
                    lastResetPasswordReqestAttempt: true,
                    emails: { select: { emailAddress: true } }
                }
            });
            if (!user) return new CustomError(CODE.ErrorUnknown);
            // If code is invalid
            if (!UserModel.validateCode(args.code, user.resetPasswordCode, user.lastResetPasswordReqestAttempt)) {
                // Generate and send new code
                await new UserModel(context.prisma).setupPasswordReset(user);
                // Return error
                return new CustomError(CODE.InvalidResetCode);
            }
            // Remove request data from user, and set new password
            await context.prisma.user.update({
                where: { id: user.id },
                data: {
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null,
                    password: UserModel.hashPassword(args.newPassword)
                }
            })
            // Return user data
            return await new UserModel(context.prisma).findById(user.id, info)
        },
        /**
         * Reports a user. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportUser: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}