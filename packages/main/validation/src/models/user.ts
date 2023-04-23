import * as yup from "yup";
import { bio, blankToUndefined, bool, email, handle, maxStrErr, name, opt, password, req, theme, transRel, YupModel, yupObj } from "../utils";
import { emailValidation } from "./email";
import { focusModeValidation } from "./focusMode";

/**
 * Schema for traditional email/password log in. NOT the form
 */
export const emailLogInSchema = yup.object().shape({
    email: opt(email),
    password: opt(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
    verificationCode: opt(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
});

export const userTranslationValidation: YupModel = transRel({
    create: {
        bio: opt(bio),
    },
    update: {
        bio: opt(bio),
    },
});

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
        ["focusModes", ["Create", "Update", "Delete"], "many", "opt", focusModeValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", userTranslationValidation],
    ], [], o),
};

/**
 * Schema for traditional email/password log in FORM.
 */
export const userDeleteOneSchema = yup.object().shape({
    password: req(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
    deletePublicData: req(bool),
});

/**
 * Schema for sending a password reset request
 */
export const emailRequestPasswordChangeSchema = yup.object().shape({
    email: req(email),
});

/**
 * Schema for resetting your password
 */
export const emailResetPasswordSchema = yup.object().shape({
    newPassword: req(password),
    confirmNewPassword: yup.string().oneOf([yup.ref("newPassword"), null], "Passwords must match"),
});

export const profileEmailUpdateValidation: YupModel<false, true> = {
    update: ({ o }) => yupObj({
        currentPassword: req(password),
        newPassword: opt(password),
    }, [
        ["emails", ["Create", "Delete"], "many", "opt", emailValidation],
    ], [], o),
};
