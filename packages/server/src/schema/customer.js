import { gql } from 'apollo-server-express';
import { TABLES } from '../db';
import bcrypt from 'bcrypt';
import { ACCOUNT_STATUS, CODE, COOKIE, joinWaitlistSchema, logInSchema, passwordSchema, signUpSchema, requestPasswordChangeSchema } from '@local/shared';
import { CustomError, validateArgs } from '../error';
import { generateToken } from '../auth';
import { customerNotifyAdmin, joinWaitlistConfirmation, sendResetPasswordLink, sendVerificationLink } from '../worker/email/queue';
import { HASHING_ROUNDS } from '../consts';
import { PrismaSelect } from '@paljs/plugins';
import { customerFromEmail, getCustomerSelect, upsertCustomer } from '../db/models/customer';

const _model = TABLES.Customer;
const LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT = 3;
const SOFT_LOCKOUT_DURATION = 15 * 60 * 1000;
const REQUEST_PASSWORD_RESET_DURATION = 2 * 24 * 3600 * 1000;
const LOGIN_ATTEMPTS_TO_HARD_LOCKOUT = 10;

export const typeDef = gql`
    enum AccountStatus {
        Deleted
        Unlocked
        SoftLock
        HardLock
    }

    input CustomerInput {
        id: ID
        firstName: String
        lastName: String
        pronouns: String
        emails: [EmailInput!]
        theme: String
        status: AccountStatus
    }

    type Customer {
        id: ID!
        firstName: String!
        lastName: String!
        fullName: String
        pronouns: String!
        emails: [Email!]!
        theme: String!
        emailVerified: Boolean!
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
            firstName: String!
            lastName: String!
            pronouns: String
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
        joinWaitlist(email: String!): Boolean
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
    AccountStatus: ACCOUNT_STATUS,
    Query: {
        customers: async (_, _args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].findMany({
                orderBy: { fullName: 'asc', },
                ...(new PrismaSelect(info).value)
            });
        },
        profile: async (_, _a, context, info) => {
            // Can only query your own profile
            const customerId = context.req.customerId;
            if (customerId === null || customerId === undefined) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].findUnique({ where: { id: customerId }, ...(new PrismaSelect(info).value) });
        }
    },
    Mutation: {
        login: async (_, args, context, info) => {
            const prismaInfo = getCustomerSelect(info);
            // If username and password wasn't passed, then use the session cookie data to validate
            if (args.username === undefined && args.password === undefined) {
                if (context.req.roles && context.req.roles.length > 0) {
                    let userData = await context.prisma[_model].findUnique({ where: { id: context.req.customerId }, ...prismaInfo });
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
                await context.prisma[_model].update({
                    where: { id: customer.id },
                    data: { resetPasswordCode: requestCode, lastResetPasswordReqestAttempt: new Date().toISOString() }
                })
                // Send new verification email
                sendResetPasswordLink(args.email, customer.id, requestCode);
                return new CustomError(CODE.MustResetPassword);
            }
            // Validate verification code, if supplied
            if (args.verificationCode === customer.id && customer.emailVerified === false) {
                customer = await context.prisma[_model].update({
                    where: { id: customer.id },
                    data: { status: ACCOUNT_STATUS.Unlocked, emailVerified: true }
                })
            }
            // Reset login attempts after 15 minutes
            const unable_to_reset = [ACCOUNT_STATUS.HardLock, ACCOUNT_STATUS.Deleted];
            if (!unable_to_reset.includes(customer.status) && Date.now() - new Date(customer.lastLoginAttempt).getTime() > SOFT_LOCKOUT_DURATION) {
                customer = await context.prisma[_model].update({
                    where: { id: customer.id },
                    data: { loginAttempts: 0 }
                })
            }
            // Before validating password, let's check to make sure the account is unlocked
            const status_to_code = {
                [ACCOUNT_STATUS.Deleted]: CODE.NoCustomer,
                [ACCOUNT_STATUS.SoftLock]: CODE.SoftLockout,
                [ACCOUNT_STATUS.HardLock]: CODE.HardLockout
            }
            if (customer.status in status_to_code) return new CustomError(status_to_code[customer.status]);
            // Now we can validate the password
            const validPassword = bcrypt.compareSync(args.password, customer.password);
            if (validPassword) {
                await generateToken(context.res, customer.id);
                await context.prisma[_model].update({
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
                return await context.prisma[_model].findUnique({ where: { id: customer.id }, ...prismaInfo });
            } else {
                let new_status = ACCOUNT_STATUS.Unlocked;
                let login_attempts = customer.loginAttempts + 1;
                if (login_attempts >= LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT) {
                    new_status = ACCOUNT_STATUS.SoftLock;
                } else if (login_attempts > LOGIN_ATTEMPTS_TO_HARD_LOCKOUT) {
                    new_status = ACCOUNT_STATUS.HardLock;
                }
                await context.prisma[_model].update({
                    where: { id: customer.id },
                    data: { status: new_status, loginAttempts: login_attempts, lastLoginAttempt: new Date().toISOString() }
                })
                return new CustomError(CODE.BadCredentials);
            }
        },
        logout: async (_, _args, context, _info) => {
            context.res.clearCookie(COOKIE.Session);
        },
        signUp: async (_, args, context, info) => {
            const prismaInfo = getCustomerSelect(info);
            // Validate input format
            const validateError = await validateArgs(signUpSchema, args);
            if (validateError) return validateError;
            // Find customer role to give to new user
            const customerRole = await context.prisma[TABLES.Role].findUnique({ where: { title: 'Customer' } });
            if (!customerRole) return new CustomError(CODE.ErrorUnknown);
            const customer = await upsertCustomer({
                prisma: context.prisma,
                info,
                data: {
                    firstName: args.firstName,
                    lastName: args.lastName,
                    pronouns: args.pronouns,
                    password: bcrypt.hashSync(args.password, HASHING_ROUNDS),
                    theme: args.theme,
                    status: ACCOUNT_STATUS.Unlocked,
                    emails: [{ emailAddress: args.email }],
                    roles: [customerRole]
                }
            })
            await generateToken(context.res, customer.id);
            // Send verification email
            sendVerificationLink(args.email, customer.id);
            // Send email to business owner
            customerNotifyAdmin(`${args.firstName} ${args.lastName}`);
            // Return cart, along with user data
            return await context.prisma[_model].findUnique({ where: { id: customer.id }, ...prismaInfo });
        },
        addCustomer: async (_, args, context, info) => {
            // Must be admin to add a customer directly
            if(!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            const prismaInfo = getCustomerSelect(info);
            // Find customer role to give to new user
            const customerRole = await context.prisma[TABLES.Role].findUnique({ where: { title: 'Customer' } });
            if (!customerRole) return new CustomError(CODE.ErrorUnknown);
            const customer = await upsertCustomer({
                prisma: context.prisma,
                info,
                data: {
                    firstName: args.input.firstName,
                    lastName: args.input.lastName,
                    pronouns: args.input.pronouns,
                    theme: 'light',
                    status: ACCOUNT_STATUS.Unlocked,
                    emails: args.input.emails,
                    roles: [customerRole]
                }
            });
            // Return cart, along with user data
            return await context.prisma[_model].findUnique({ where: { id: customer.id }, ...prismaInfo });
        },
        updateCustomer: async (_, args, context, info) => {
            // Must be admin, or updating your own
            if(!context.req.isAdmin && (context.req.customerId !== args.input.id)) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let customer = await context.prisma[_model].findUnique({ 
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
        deleteCustomer: async (_, args, context) => {
            // Must be admin, or deleting your own
            if(!context.req.isAdmin && (context.req.customerId !== args.input.id)) return new CustomError(CODE.Unauthorized);
            // Check for correct password
            let customer = await context.prisma[_model].findUnique({ 
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
            await context.prisma[_model].delete({ where: { id: customer.id } });
            return true;
        },
        joinWaitlist: async (_, args, context) => {
            // Validate input format
            const validateError = await validateArgs(joinWaitlistSchema, args);
            if (validateError) return validateError;
            // Validate email address
            const emailRow = await context.prisma[TABLES.Email].findUnique({ where: { emailAddress: args.email } });
            if (emailRow) throw new CustomError(CODE.EmailInUse);
            // Generate confirmation code
            const confirmationCode = bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '');
            // Create new user
            const customerRole = await context.prisma[TABLES.Role].findUnique({ where: { title: 'Customer' } });
            if (!customerRole) return new CustomError(CODE.ErrorUnknown);
            await upsertCustomer({
                prisma: context.prisma,
                data: {
                    firstName: '',
                    lastName: '',
                    pronouns: '',
                    theme: 'light',
                    confirmationCode,
                    confirmationCodeDate: new Date().toISOString(),
                    status: ACCOUNT_STATUS.Unlocked,
                    emails: args.email,
                    roles: [customerRole]
                }
            });
            // Send confirmation email
            joinWaitlistConfirmation(args.email);
            return true;
        },
        requestPasswordChange: async (_, args, context) => {
            // Validate input format
            const validateError = await validateArgs(requestPasswordChangeSchema, args);
            if (validateError) return validateError;
            // Find customer in database
            const customer = await customerFromEmail(args.email, context.prisma);
            // Generate request code
            const requestCode = bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '');
            // Store code and request time in customer row
            await context.prisma[_model].update({
                where: { id: customer.id },
                data: { resetPasswordCode: requestCode, lastResetPasswordReqestAttempt: new Date().toISOString() }
            })
            // Send email with correct reset link
            sendResetPasswordLink(args.email, customer.id, requestCode);
            return true;
        },
        resetPassword: async(_, args, context, info) => {
            // Validate input format
            const validateError = await validateArgs(passwordSchema, args.newPassword);
            if (validateError) return validateError;
            // Find customer in database
            const customer = await context.prisma[_model].findUnique({ 
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
                await context.prisma[_model].update({
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
            await context.prisma[_model].update({
                where: { id: customer.id },
                data: { 
                    resetPasswordCode: null, 
                    lastResetPasswordReqestAttempt: null,
                    password: bcrypt.hashSync(args.newPassword, HASHING_ROUNDS)
                }
            })
            // Return customer data
            const prismaInfo = getCustomerSelect(info);
            return await context.prisma[TABLES.Customer].findUnique({ where: { id: customer.id }, ...prismaInfo });
        },
        changeCustomerStatus: async (_, args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            await context.prisma[_model].update({
                where: { id: args.id },
                data: { status: args.status }
            })
            return true;
        },
        addCustomerRole: async (_, args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            await context.prisma[TABLES.CustomerRoles].create({ data: { 
                customerId: args.id,
                roleId: args.roleId
            } })
            return await context.prisma[_model].findUnique({ where: { id: args.id }, ...(new PrismaSelect(info).value) });
        },
        removeCustomerRole: async (_, args, context) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[TABLES.CustomerRoles].delete({ where: { 
                customerId: args.id,
                roleId: args.roleId
            } })
        },
    }
}