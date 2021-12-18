import { gql } from 'apollo-server-express';
import { CODE, COOKIE, logInSchema, passwordSchema, signUpSchema, requestPasswordChangeSchema, ROLES } from '@local/shared';
import { CustomError, validateArgs } from '../error';
import { generateToken } from '../auth/auth';
import { UserModel } from '../models';
import { DeleteUserInput, LogInInput, ReportInput, RequestPasswordChangeInput, ResetPasswordInput, SignUpInput, UpdateUserInput, User } from './types';
import { IWrap } from '../types';
import { Context } from '../context';
import pkg from '@prisma/client';
import { GraphQLResolveInfo } from 'graphql';
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
        password: String!
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
        profile: async (_parent: undefined, _args: undefined, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<User | null> => {
            // TODO add restrictions to what data can be queried
            return await UserModel(prisma).findById({ id: req.userId ?? '' }, info);
        }
    },
    Mutation: {
        logIn: async (_parent: undefined, { input }: IWrap<LogInInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<User> => {
            // If email and password are not provided, attempt to log in using session cookie
            if (!Boolean(input.email) && !Boolean(input.password)) {
                // If session is expired
                if (!Array.isArray(req.roles) || req.roles.length === 0) {
                    res.clearCookie(COOKIE.Session);
                    throw new CustomError(CODE.SessionExpired);
                }
                const userData = await UserModel(prisma).findById({ id: req.userId ?? '' }, info);
                if (userData) return userData;
                // If user data failed to fetch, clear session and return error
                res.clearCookie(COOKIE.Session);
                throw new CustomError(CODE.ErrorUnknown);
            }
            // Validate arguments with schema
            const validateError = await validateArgs(logInSchema, input);
            if (validateError) throw validateError;
            // Get user
            let user = await UserModel(prisma).findByEmail(input.email as string);
            // Check for password in database, if doesn't exist, send a password reset link
            // TODO fix this. Can't have password in type, but need password returned
            // if (!Boolean(user.password)) {
            //     await UserModel(prisma).setupPasswordReset(user);
            //     throw new CustomError(CODE.MustResetPassword);
            // }
            // Validate verification code, if supplied
            if (Boolean(input.verificationCode)) {
                user = await UserModel(prisma).validateVerificationCode(user, input.verificationCode as string);
            }
            user = await UserModel(prisma).logIn(input.password as string, user, info);
            if (Boolean(user)) {
                // Set session token
                await generateToken(res, user.id);
                return user;
            } else {
                throw new CustomError(CODE.BadCredentials);
            }
        },
        logOut: async (_parent: undefined, _args: undefined, { res }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            res.clearCookie(COOKIE.Session);
            return true;
        },
        signUp: async (_parent: undefined, { input }: IWrap<SignUpInput>, { prisma, res }: Context, info: GraphQLResolveInfo): Promise<User> => {
            // Validate input format
            const validateError = await validateArgs(signUpSchema, input);
            if (validateError) throw validateError;
            // Find user role to give to new user
            const actorRole = await prisma.role.findUnique({ where: { title: ROLES.Actor } });
            if (!actorRole) throw new CustomError(CODE.ErrorUnknown);
            // Create user object
            const user = await UserModel(prisma).upsertUser({
                username: input.username,
                password: UserModel(prisma).hashPassword(input.password),
                theme: input.theme,
                status: AccountStatus.UNLOCKED,
                emails: [{ emailAddress: input.email }],
                roles: [actorRole]
            }, info);
            // Set up session token
            await generateToken(res, user.id);
            // Send verification email
            await UserModel(prisma).setupVerificationCode(user);
            // Return user data
            return await UserModel(prisma).findById({ id: user.id }, info) as User;
        },
        updateUser: async (_parent: undefined, { input }: IWrap<UpdateUserInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<User> => {
            // Must be updating your own
            if (req.userId !== input.data.id) throw new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await prisma.user.findUnique({ where: { id: input.data.id } });
            if (!user) throw new CustomError(CODE.InvalidArgs);
            if (!UserModel(prisma).validatePassword(input.currentPassword, user)) throw new CustomError(CODE.BadCredentials);
            // Update user
            return await UserModel(prisma).upsertUser(input.data, info);
        },
        deleteUser: async (_parent: undefined, { input }: IWrap<DeleteUserInput>, { prisma, req }: any, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be deleting your own
            if (req.userId !== input.id) throw new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await prisma.user.findUnique({ where: { id: input.id } });
            if (!user) throw new CustomError(CODE.InvalidArgs);
            if (!UserModel(prisma).validatePassword(input.password, user)) throw new CustomError(CODE.BadCredentials);
            // Delete user
            return await UserModel(prisma).delete(user.id);
        },
        requestPasswordChange: async (_parent: undefined, { input }: IWrap<RequestPasswordChangeInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Validate input format
            const validateError = await validateArgs(requestPasswordChangeSchema, input);
            if (validateError) throw validateError;
            // Find user in database
            const user = await UserModel(prisma).findByEmail(input.email);
            // Generate and send password reset code
            return await UserModel(prisma).setupPasswordReset(user);
        },
        resetPassword: async (_parent: undefined, { input }: IWrap<ResetPasswordInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<User> => {
            // Validate input format
            const validateError = await validateArgs(passwordSchema, input.newPassword);
            if (validateError) throw validateError;
            // Find user in database
            let user = await prisma.user.findUnique({
                where: { id: input.id },
                select: {
                    id: true,
                    status: true,
                    resetPasswordCode: true,
                    lastResetPasswordReqestAttempt: true,
                    emails: { select: { emailAddress: true } }
                }
            });
            if (!user) throw new CustomError(CODE.ErrorUnknown);
            // If code is invalid
            if (!UserModel(prisma).validateCode(input.code, user.resetPasswordCode ?? '', user.lastResetPasswordReqestAttempt as Date)) {
                // Generate and send new code
                await UserModel(prisma).setupPasswordReset(user);
                // Return error
                throw new CustomError(CODE.InvalidResetCode);
            }
            // Remove request data from user, and set new password
            await prisma.user.update({
                where: { id: user.id },
                data: {
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null,
                    password: UserModel(prisma).hashPassword(input.newPassword)
                }
            })
            // Return user data
            return await UserModel(prisma).findById({ id: user.id }, info) as User;
        },
        /**
         * Reports a user. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportUser: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await UserModel(prisma).report(input);
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, organizations, etc.).
         * In the future, an import function will be added.
         * @returns JSON of all user data
         */
        exportData: async (_parent: undefined, _args: undefined, context: Context, _info: GraphQLResolveInfo): Promise<string> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}