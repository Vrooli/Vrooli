import { gql } from 'apollo-server-express';
import bcrypt from 'bcrypt';
import { CODE, COOKIE, joinWaitlistSchema, logInSchema, passwordSchema, profileSchema, signUpSchema, requestPasswordChangeSchema } from '@local/shared';
import { CustomError, validateArgs } from '../error';
import { generateToken } from '../auth';
import { confirmJoinWaitlist, customerNotifyAdmin, joinedWaitlist, joinWaitlistNotifyAdmin, sendResetPasswordLink, sendVerificationLink } from '../worker/email/queue';
import { PrismaSelect } from '@paljs/plugins';
import { customerFromEmail, getCustomerSelect, upsertCustomer } from '../db/models/customer';
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
        emails: [EmailInput!]
        theme: String
        status: AccountStatus
    }

    type Customer {
        id: ID!
        username: String!
        emails: [Email!]!
        theme: String!
        status: AccountStatus!
        roles: [CustomerRole!]!
        feedback: [Feedback!]!
    }

    extend type Query {
        customers: [Customer!]!
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
            email: String!
            theme: String!
            marketingEmails: Boolean!
            password: String!
        ): Customer!
        addCustomer(input: CustomerInput!): Customer!
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
        requestPasswordChange(email: String!): Boolean
        resetPassword(
            id: ID!
            code: String!
            newPassword: String!
        ): Customer!
        changeCustomerStatus(
            id: ID!
            status: AccountStatus!
        ): Boolean
        addCustomerRole(
            id: ID!
            roleId: ID!
        ): Customer!
        removeCustomerRole(
            id: ID!
            roleId: ID!
        ): Count!
    }
`

export const resolvers = {
    AccountStatus: AccountStatus,
    Query: {
        customers: async (_parent, _args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma.customer.findMany({
                orderBy: { username: 'asc', },
                ...(new PrismaSelect(info).value)
            });
        },
        profile: async (_parent, _args, context, info) => {
            // Can only query your own profile
            const customerId = context.req.customerId;
            if (customerId === null || customerId === undefined) return new CustomError(CODE.Unauthorized);
            return await context.prisma.customer.findUnique({ where: { id: customerId }, ...(new PrismaSelect(info).value) });
        }
    },
    Mutation: {
        login: async (_parent, args, context, info) => {
            // TEMP UNTIL WAITLIST IS GONE - only allow admin to log in
            if (!context.req.isAdmin && process.env.ADMIN_EMAIL !== args.email) return new CustomError(CODE.Unauthorized);
            const prismaInfo = getCustomerSelect(info);
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
            if (args.verificationCode === customer.id) {
                customer = await context.prisma.customer.update({
                    where: { id: customer.id },
                    data: { status: AccountStatus.UNLOCKED }
                })
                await context.prisma.email.update({ 
                    where: { emailAddress: args.email },
                    data: { verified: true }
                });
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
            const status_to_code = {
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
                // Return cart, along with user data
                return await context.prisma.customer.findUnique({ where: { id: customer.id }, ...prismaInfo });
            } else {
                let new_status = AccountStatus.UNLOCKED;
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
        logout: async (_parent, _args, context, _info) => {
            context.res.clearCookie(COOKIE.Session);
        },
        signUp: async (_parent, args, context, info) => {
            const prismaInfo = getCustomerSelect(info);
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
            // Return cart, along with user data
            return await context.prisma.customer.findUnique({ where: { id: customer.id }, ...prismaInfo });
        },
        addCustomer: async (_parent, args, context, info) => {
            // Must be admin to add a customer directly
            if(!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            const prismaInfo = getCustomerSelect(info);
            // Find customer role to give to new user
            const customerRole = await context.prisma.role.findUnique({ where: { title: 'Customer' } });
            if (!customerRole) return new CustomError(CODE.ErrorUnknown);
            const customer = await upsertCustomer({
                prisma: context.prisma,
                info,
                data: {
                    username: args.input.username,
                    theme: 'light',
                    status: AccountStatus.UNLOCKED,
                    emails: args.input.emails,
                    roles: [customerRole]
                }
            });
            // Return cart, along with user data
            return await context.prisma.customer.findUnique({ where: { id: customer.id }, ...prismaInfo });
        },
        updateCustomer: async (_parent, args, context, info) => {
            // Must be admin, or updating your own
            if(!context.req.isAdmin && (context.req.customerId !== args.input.id)) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let customer = await context.prisma.customer.findUnique({ 
                where: { id: args.input.id },
                select: {
                    id: true,
                    password: true,
                }
            });
            if(!bcrypt.compareSync(args.currentPassword, customer.password)) return new CustomError(CODE.BadCredentials);
            // Validate input format
            const validateError = await validateArgs(profileSchema, args.input);
            if (validateError) return validateError;
            const user = await upsertCustomer({
                prisma: context.prisma,
                info,
                data: args.input
            })
            return user;
        },
        deleteCustomer: async (_parent, args, context, _info) => {
            // Must be admin, or deleting your own
            if(!context.req.isAdmin && (context.req.customerId !== args.input.id)) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let customer = await context.prisma.customer.findUnique({ 
                where: { id: args.id },
                select: {
                    id: true,
                    password: true
                }
            });
            if (!customer) return new CustomError(CODE.ErrorUnknown);
            // If admin, make sure you are not deleting yourself
            if (context.req.isAdmin) {
                if (customer.id === context.req.customerId) return new CustomError(CODE.CannotDeleteYourself);
            }
            // If not admin, make sure correct password is entered
            else if (!context.req.isAdmin) {
                if(!bcrypt.compareSync(args.password, customer.password)) return new CustomError(CODE.BadCredentials);
            }
            // Delete account
            await context.prisma.customer.delete({ where: { id: customer.id } });
            return true;
        },
        joinWaitlist: async (_parent, args, context, _info) => {
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
        verifyWaitlist: async (_parent, args, context, _info) => {
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
        requestPasswordChange: async (_parent, args, context, _info) => {
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
        resetPassword: async(_parent, args, context, info) => {
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
            const prismaInfo = getCustomerSelect(info);
            return await context.prisma.customer.findUnique({ where: { id: customer.id }, ...prismaInfo });
        },
        changeCustomerStatus: async (_parent, args, context, _info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            await context.prisma.customer.update({
                where: { id: args.id },
                data: { status: args.status }
            })
            return true;
        },
        addCustomerRole: async (_parent, args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            await context.prisma.customer_roles.create({ data: { 
                customerId: args.id,
                roleId: args.roleId
            } })
            return await context.prisma.customer.findUnique({ where: { id: args.id }, ...(new PrismaSelect(info).value) });
        },
        removeCustomerRole: async (_parent, args, context, _info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma.customer_roles.delete({ where: { 
                customerId: args.id,
                roleId: args.roleId
            } })
        },
    }
}