import { id, bio, name, handle, maxStrErr, blankToUndefined, email, req, opt, YupModel, transRel, rel, theme, password } from '../utils';
import * as yup from 'yup';
import { userScheduleValidation } from './userSchedule';

export const emailSchema = yup.object().shape({
    emailAddress: req(email),
    receivesAccountUpdates: opt(yup.bool().default(true)),
    receivesBusinessUpdates: opt(yup.bool().default(true)),
    userId: req(id),
});

/**
 * Schema for traditional email/password log in. NOT the form
 */
export const emailLogInSchema = yup.object().shape({
    email: opt(email),
    password: opt(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
    verificationCode: opt(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
})

export const userTranslationValidation: YupModel = transRel({
    create: {
        bio: opt(bio),
    },
    update: {
        bio: opt(bio),
    },
})

export const userValidation: YupModel<false, true> = {
    // Can't create a user directly - must use sign up form(s)
    update: () => yup.object().shape({
        handle: opt(handle),
        name: opt(name),
        theme: opt(theme),
        isPrivate: opt(yup.boolean()),
        isPrivateApis: opt(yup.boolean()),
        isPrivateApisCreated: opt(yup.boolean()),
        isPrivateMemberships: opt(yup.boolean()),
        isPrivateOrganizationsCreated: opt(yup.boolean()),
        isPrivateProjects: opt(yup.boolean()),
        isPrivateProjectsCreated: opt(yup.boolean()),
        isPrivatePullRequests: opt(yup.boolean()),
        isPrivateQuestionsAnswered: opt(yup.boolean()),
        isPrivateQuestionsAsked: opt(yup.boolean()),
        isPrivateQuizzesCreated: opt(yup.boolean()),
        isPrivateRoles: opt(yup.boolean()),
        isPrivateRoutines: opt(yup.boolean()),
        isPrivateRoutinesCreated: opt(yup.boolean()),
        isPrivateSmartContracts: opt(yup.boolean()),
        isPrivateStandards: opt(yup.boolean()),
        isPrivateStandardsCreated: opt(yup.boolean()),
        isPrivateStars: opt(yup.boolean()),
        isPrivateVotes: opt(yup.boolean()),
        ...rel('schedules', ['Create', 'Update', 'Delete'], 'many', 'opt', userScheduleValidation),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', userTranslationValidation),
    }),
}

/**
 * Schema for traditional email/password log in FORM.
 */
export const userDeleteOneSchema = yup.object().shape({
    password: req(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
    deletePublicData: req(yup.boolean()),
})

/**
 * Schema for sending a password reset request
 */
export const emailRequestPasswordChangeSchema = yup.object().shape({
    email: req(email)
})

/**
 * Schema for resetting your password
 */
export const emailResetPasswordSchema = yup.object().shape({
    newPassword: req(password),
    confirmNewPassword: yup.string().oneOf([yup.ref('newPassword'), null], 'Passwords must match')
})