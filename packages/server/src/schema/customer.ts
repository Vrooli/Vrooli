import { gql } from 'apollo-server-express';
import bcrypt from 'bcrypt';
import { CODE, COOKIE, joinWaitlistSchema, logInSchema, passwordSchema, signUpSchema, requestPasswordChangeSchema } from '@local/shared';
import { CustomError, validateArgs } from '../error';
import { generateToken } from '../auth/auth';
import { confirmJoinWaitlist, customerNotifyAdmin, joinedWaitlist, joinWaitlistNotifyAdmin, sendResetPasswordLink, sendVerificationLink } from '../worker/email/queue';
import { PrismaSelect } from '@paljs/plugins';
import { customerFromEmail, upsertCustomer } from '../db/models/customer';
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

    input CustomerInput {
        id: ID
        username: String
        pronouns: String
        emails: [EmailInput!]
        theme: String
        status: AccountStatus
    }

    type Customer {
        id: ID!
        username: String!
        pronouns: String!
        emails: [Email!]!
        theme: String!
        emailVerified: Boolean!
        status: AccountStatus!
        roles: [CustomerRole!]!
    }

    extend type Query {
        profile: Customer!
    }

    extend type Mutation {
        login(
            email: String
            password: String,
            verificationCode: String
        ): Customer!
        logout: Boolean
        signUp(
            username: String!
            pronouns: String
            email: String!
            theme: String!
            marketingEmails: Boolean!
            password: String!
        ): Customer!
        updateCustomer(
            input: CustomerInput!
            currentPassword: String!
            newPassword: String
        ): Customer!
        deleteCustomer(
            id: ID!
            password: String
        ): Boolean
        joinWaitlist(
            username: String!
            email: String!
        ): Boolean
        verifyWaitlist(
            confirmationCode: String!
        ): Boolean
        requestPasswordChange(
            email: String!
        ): Boolean
        resetPassword(
            id: ID!
            code: String!
            newPassword: String!
        ): Customer!
    }
`

export const resolvers = {
    AccountStatus: AccountStatus,
    Query: {
        profile: async (_parent: undefined, _args: any, context: any, info: any) => {
            // Can only query your own profile
            const customerId = context.req.customerId;
            if (customerId === null || customerId === undefined) return new CustomError(CODE.Unauthorized);
            return await context.prisma.customer.findUnique({ where: { id: customerId }, ...(new PrismaSelect(info).value) });
        }
    },
    Mutation: {
        login: async (_parent: undefined, args: any, context: any, info: any) => {
            const prismaInfo = new PrismaSelect(info).value
            // If username and password wasn't passed, then use the session cookie data to validate
            if (args.username === undefined && args.password === undefined) {
                if (context.req.roles && context.req.roles.length > 0) {
                    let userData = await context.prisma.customer.findUnique({ where: { id: context.req.customerId }, ...prismaInfo });
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
            // Get customer
            let customer = await customerFromEmail(args.email, context.prisma);
            // Check for password in database, if doesn't exist, send a password reset link
            if (!customer.password) {
                // Generate new code
                const requestCode = bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '');
                // Store code and request time in customer row
                await context.prisma.customer.update({
                    where: { id: customer.id },
                    data: { resetPasswordCode: requestCode, lastResetPasswordReqestAttempt: new Date().toISOString() }
                })
                // Send new verification email
                sendResetPasswordLink(args.email, customer.id, requestCode);
                return new CustomError(CODE.MustResetPassword);
            }
            // Validate verification code, if supplied
            if (args.verificationCode === customer.id && customer.emailVerified === false) {
                customer = await context.prisma.customer.update({
                    where: { id: customer.id },
                    data: { status: AccountStatus.UNLOCKED, emailVerified: true }
                })
            }
            // Reset login attempts after 15 minutes
            const unable_to_reset = [AccountStatus.HARD_LOCKED, AccountStatus.DELETED];
            if (!unable_to_reset.includes(customer.status) && Date.now() - new Date(customer.lastLoginAttempt).getTime() > SOFT_LOCKOUT_DURATION) {
                customer = await context.prisma.customer.update({
                    where: { id: customer.id },
                    data: { loginAttempts: 0 }
                })
            }
            // Before validating password, let's check to make sure the account is unlocked
            const status_to_code: any = {
                [AccountStatus.DELETED]: CODE.NoCustomer,
                [AccountStatus.SOFT_LOCKED]: CODE.SoftLockout,
                [AccountStatus.HARD_LOCKED]: CODE.HardLockout
            }
            if (customer.status in status_to_code) return new CustomError(status_to_code[customer.status]);
            // Now we can validate the password
            const validPassword = bcrypt.compareSync(args.password, customer.password);
            if (validPassword) {
                await generateToken(context.res, customer.id);
                await context.prisma.customer.update({
                    where: { id: customer.id },
                    data: { 
                        loginAttempts: 0, 
                        lastLoginAttempt: new Date().toISOString(), 
                        resetPasswordCode: null, 
                        lastResetPasswordReqestAttempt: null 
                    },
                    ...prismaInfo
                })
                // Return user data
                return await context.prisma.customer.findUnique({ where: { id: customer.id }, ...prismaInfo });
            } else {
                let new_status: any = AccountStatus.UNLOCKED;
                let login_attempts = customer.loginAttempts + 1;
                if (login_attempts >= LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT) {
                    new_status = AccountStatus.SOFT_LOCKED;
                } else if (login_attempts > LOGIN_ATTEMPTS_TO_HARD_LOCKOUT) {
                    new_status = AccountStatus.HARD_LOCKED;
                }
                await context.prisma.customer.update({
                    where: { id: customer.id },
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
            // Find customer role to give to new user
            const customerRole = await context.prisma.role.findUnique({ where: { title: 'Customer' } });
            if (!customerRole) return new CustomError(CODE.ErrorUnknown);
            const customer = await upsertCustomer({
                prisma: context.prisma,
                info,
                data: {
                    username: args.username,
                    password: bcrypt.hashSync(args.password, HASHING_ROUNDS),
                    theme: args.theme,
                    status: AccountStatus.UNLOCKED,
                    emails: [{ emailAddress: args.email }],
                    roles: [customerRole]
                }
            })
            await generateToken(context.res, customer.id);
            // Send verification email
            sendVerificationLink(args.email, customer.id);
            // Send email to business owner
            customerNotifyAdmin(args.username);
            // Return user data
            return await context.prisma.customer.findUnique({ where: { id: customer.id }, ...prismaInfo });
        },
        updateCustomer: async (_parent: undefined, args: any, context: any, info: any) => {
            // Must be updating your own
            if(context.req.customerId !== args.input.id) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let customer = await context.prisma.customer.findUnique({ 
                where: { id: args.input.id },
                select: {
                    id: true,
                    password: true,
                }
            });
            if(!bcrypt.compareSync(args.currentPassword, customer.password)) return new CustomError(CODE.BadCredentials);
            const user = await upsertCustomer({
                prisma: context.prisma,
                info,
                data: args.input
            })
            return user;
        },
        deleteCustomer: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must be deleting your own
            if(context.req.customerId !== args.input.id) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let customer = await context.prisma.customer.findUnique({ 
                where: { id: args.id },
                select: {
                    id: true,
                    password: true
                }
            });
            if (!customer) return new CustomError(CODE.ErrorUnknown);
            // Make sure correct password is entered
            if(!bcrypt.compareSync(args.password, customer.password)) return new CustomError(CODE.BadCredentials);
            // Delete account
            await context.prisma.customer.delete({ where: { id: customer.id } });
            return true;
        },
        joinWaitlist: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Validate input format
            const validateError = await validateArgs(joinWaitlistSchema, args);
            if (validateError) return validateError;
            // Validate email address
            const emailRow = await context.prisma.email.findUnique({ where: { emailAddress: args.email } });
            if (emailRow) return new CustomError(CODE.EmailInUse);
            // Generate confirmation code
            const confirmationCode = bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '');
            // Create new user
            const customerRole = await context.prisma.role.findUnique({ where: { title: 'Customer' } });
            if (!customerRole) return new CustomError(CODE.ErrorUnknown);
            await upsertCustomer({
                prisma: context.prisma,
                data: {
                    username: args.username,
                    theme: 'light',
                    confirmationCode,
                    confirmationCodeDate: new Date().toISOString(),
                    status: AccountStatus.UNLOCKED,
                    emails: [{ emailAddress: args.email }],
                    roles: [customerRole]
                }
            });
            // Send confirmation email
            confirmJoinWaitlist(args.email, confirmationCode);
            return true;
        },
        verifyWaitlist: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Find customer
            const customer = await context.prisma.customer.findUnique({ 
                where: { confirmationCode: args.confirmationCode },
                select: { username: true, emails: { select: { emailAddress: true, verified: true } } }
            });
            if (!customer || customer.emails.length === 0) throw new CustomError(CODE.ErrorUnknown);
            if (!customer.emails[0].verified) {
                await context.prisma.email.update({ 
                    where: { emailAddress: customer.emails[0].emailAddress },
                    data: { verified: true }
                });
                joinedWaitlist(customer.emails[0].emailAddress);
                joinWaitlistNotifyAdmin(customer.username);
            }
            return true;
        },
        requestPasswordChange: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Validate input format
            const validateError = await validateArgs(requestPasswordChangeSchema, args);
            if (validateError) return validateError;
            // Find customer in database
            const customer = await customerFromEmail(args.email, context.prisma);
            // Generate request code
            const requestCode = bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '');
            // Store code and request time in customer row
            await context.prisma.customer.update({
                where: { id: customer.id },
                data: { resetPasswordCode: requestCode, lastResetPasswordReqestAttempt: new Date().toISOString() }
            })
            // Send email with correct reset link
            sendResetPasswordLink(args.email, customer.id, requestCode);
            return true;
        },
        resetPassword: async(_parent: undefined, args: any, context: any, info: any) => {
            // Validate input format
            const validateError = await validateArgs(passwordSchema, args.newPassword);
            if (validateError) return validateError;
            // Find customer in database
            const customer = await context.prisma.customer.findUnique({ 
                where: { id: args.id },
                select: {
                    id: true,
                    resetPasswordCode: true,
                    lastResetPasswordReqestAttempt: true,
                    emails: { select: { emailAddress: true } }
                }
            });
            if (!customer) return new CustomError(CODE.ErrorUnknown);
            // Verify request code and that request was made within 48 hours
            if (!customer.resetPasswordCode ||
                customer.resetPasswordCode !== args.code ||
                Date.now() - new Date(customer.lastResetPasswordReqestAttempt).getTime() > REQUEST_PASSWORD_RESET_DURATION) {
                // Generate new code
                const requestCode = bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '');
                // Store code and request time in customer row
                await context.prisma.customer.update({
                    where: { id: customer.id },
                    data: { resetPasswordCode: requestCode, lastResetPasswordReqestAttempt: new Date().toISOString() }
                })
                // Send new verification email
                for (const email of customer.emails) {
                    sendResetPasswordLink(email.emailAddress, customer.id, requestCode);
                }
                // Return error
                return new CustomError(CODE.InvalidResetCode);
            } 
            // Remove request data from customer, and set new password
            await context.prisma.customer.update({
                where: { id: customer.id },
                data: { 
                    resetPasswordCode: null, 
                    lastResetPasswordReqestAttempt: null,
                    password: bcrypt.hashSync(args.newPassword, HASHING_ROUNDS)
                }
            })
            // Return customer data
            const prismaInfo = new PrismaSelect(info).value
            return await context.prisma.customer.findUnique({ where: { id: customer.id }, ...prismaInfo });
        },
    }
}