import { id, bio, name, handle, maxStrErr, blankToUndefined, email, req, opt, YupModel, transRel, theme, password, yupObj, bool } from '../utils';
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
    update: ({ o }) => yupObj({
        handle: opt(handle),
        name: opt(name),
        theme: opt(theme),
        isPrivate: opt(bool),
        isPrivateApis: opt(bool),
        isPrivateApisCreated: opt(bool),
        isPrivateMemberships: opt(bool),
        isPrivateOrganizationsCreated: opt(bool),
        isPrivateProjects: opt(bool),
        isPrivateProjectsCreated: opt(bool),
        isPrivatePullRequests: opt(bool),
        isPrivateQuestionsAnswered: opt(bool),
        isPrivateQuestionsAsked: opt(bool),
        isPrivateQuizzesCreated: opt(bool),
        isPrivateRoles: opt(bool),
        isPrivateRoutines: opt(bool),
        isPrivateRoutinesCreated: opt(bool),
        isPrivateSmartContracts: opt(bool),
        isPrivateStandards: opt(bool),
        isPrivateStandardsCreated: opt(bool),
        isPrivateBookmarks: opt(bool),
        isPrivateVotes: opt(bool),
    }, [
        ['schedules', ['Create', 'Update', 'Delete'], 'many', 'opt', userScheduleValidation],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', userTranslationValidation],
    ], [], o),
}

/**
 * Schema for traditional email/password log in FORM.
 */
export const userDeleteOneSchema = yup.object().shape({
    password: req(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
    deletePublicData: req(bool),
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