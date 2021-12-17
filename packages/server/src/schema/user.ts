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

    input LogInInput {
        email: String
        password: String
        verificationCode: String
    }

    input SignUpInput {
        username: String!
        pronouns: String
        email: String!
        theme: String!
        marketingEmails: Boolean!
        password: String!
    }

    input UpdateUserInput {
        data: UserInput!
        currentPassword: String!
        newPassword: String
    }

    input DeleteUserInput {
        id: ID!
        password: String
    }

    input RequestPasswordChangeInput {
        email: String!
    }

    input ResetPasswordInput {
        id: ID!
        code: String!
        newPassword: String!
    }

    extend type Query {
        profile: User!
    }

    extend type Mutation {
        logIn(input: LogInInput!): User!
        logOut: Boolean
        signUp(input: SignUpInput!): User!
        updateUser(input: UpdateUserInput!): User!
        deleteUser(input: DeleteUserInput!): Boolean
        requestPasswordChange(input: RequestPasswordChangeInput!): Boolean
        resetPassword(input: ResetPasswordInput!): User!
        reportUser(input: ReportInput!): Boolean!
        exportData: String!
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
        logIn: async (_parent: undefined, { input }: any, context: any, info: any) => {
            const userModel = new UserModel(context.prisma);
            // If username and password are not provided, attempt to log in using session cookie
            if (!Boolean(input.username) && !Boolean(input.password)) {
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
            const validateError = await validateArgs(logInSchema, input);
            if (validateError) return validateError;
            // Get user
            let user = await new UserModel(context.prisma).fromEmail(input.email);
            // Check for password in database, if doesn't exist, send a password reset link
            if (!Boolean(user.password)) {
                await new UserModel(context.prisma).setupPasswordReset(user);
                return new CustomError(CODE.MustResetPassword);
            }
            // Validate verification code, if supplied
            if (Boolean(input.verificationCode)) {
                user = await new UserModel(context.prisma).validateVerificationCode(user, input.verificationCode);
            }
            user = await userModel.logIn(input.password, user, info);
            if (Boolean(user)) {
                // Set session token
                await generateToken(context.res, user.id);
                return user;
            } else {
                return new CustomError(CODE.BadCredentials);
            }
        },
        logOut: async (_parent: undefined, _args: any, context: any, _info: any) => {
            context.res.clearCookie(COOKIE.Session);
        },
        signUp: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Validate input format
            const validateError = await validateArgs(signUpSchema, input);
            if (validateError) return validateError;
            // Find user role to give to new user
            const actorRole = await context.prisma.role.findUnique({ where: { title: ROLES.Actor } });
            if (!actorRole) return new CustomError(CODE.ErrorUnknown);
            // Create user object
            const user = await new UserModel(context.prisma).upsertUser({
                username: input.username,
                password: UserModel.hashPassword(input.password),
                theme: input.theme,
                status: AccountStatus.UNLOCKED,
                emails: [{ emailAddress: input.email }],
                roles: [actorRole]
            }, info);
            // Set up session token
            await generateToken(context.res, user.id);
            // Send verification email
            await new UserModel(context.prisma).setupVerificationCode(user);
            // Return user data
            return await new UserModel(context.prisma).findById(user.id, info)
        },
        updateUser: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be updating your own
            if (context.req.userId !== input.id) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await context.prisma.user.findUnique({ where: { id: input.id } });
            if (!user) return new CustomError(CODE.InvalidArgs);
            if (!UserModel.validatePassword(input.currentPassword, user)) return new CustomError(CODE.BadCredentials);
            // Update user
            return await new UserModel(context.prisma).upsertUser(input.data, info);
        },
        deleteUser: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be deleting your own
            if (context.req.userId !== input.id) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await context.prisma.user.findUnique({ where: { id: input.id } });
            if (!user) return new CustomError(CODE.InvalidArgs);
            if (!UserModel.validatePassword(input.password, user)) return new CustomError(CODE.BadCredentials);
            // Delete user
            await new UserModel(context.prisma).delete(user.id);
            return true;
        },
        requestPasswordChange: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Validate input format
            const validateError = await validateArgs(requestPasswordChangeSchema, input);
            if (validateError) return validateError;
            // Find user in database
            const user = await new UserModel(context.prisma).fromEmail(input.email);
            // Generate and send password reset code
            await new UserModel(context.prisma).setupPasswordReset(user);
            return true;
        },
        resetPassword: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Validate input format
            const validateError = await validateArgs(passwordSchema, input.newPassword);
            if (validateError) return validateError;
            // Find user in database
            let user = await context.prisma.user.findUnique({
                where: { id: input.id },
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
            if (!UserModel.validateCode(input.code, user.resetPasswordCode, user.lastResetPasswordReqestAttempt)) {
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
                    password: UserModel.hashPassword(input.newPassword)
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
         reportUser: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, organizations, etc.).
         * In the future, an import function will be added.
         * @returns JSON of all user data
         */
        exportData: async (_parent: undefined, _args: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}