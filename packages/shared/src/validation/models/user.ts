import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bio, bool, email, handle, id, imageFile, name, password, theme } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { botSettings, botValidation } from "./bot.js";
import { emailValidation } from "./email.js";
import { focusModeValidation } from "./focusMode.js";

/**
 * Schema for traditional email/password log in. NOT the form
 */
export const emailLogInSchema = yup.object().shape({
    email: opt(email),
    password: opt(yup.string().trim().removeEmptyString().max(128, maxStrErr)),
    verificationCode: opt(yup.string().trim().removeEmptyString().max(128, maxStrErr)),
});

export const userTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        bio: opt(bio),
    }),
    update: () => ({
        bio: opt(bio),
    }),
});

export const profileValidation: YupModel<["update"]> = {
    // Can't create a non-bot user directly - must use sign up form(s)
    update: (d) => yupObj({
        bannerImage: opt(imageFile),
        handle: opt(handle),
        name: opt(name),
        isPrivate: opt(bool),
        isPrivateApis: opt(bool),
        isPrivateApisCreated: opt(bool),
        isPrivateCodes: opt(bool),
        isPrivateCodesCreated: opt(bool),
        isPrivateMemberships: opt(bool),
        isPrivateProjects: opt(bool),
        isPrivateProjectsCreated: opt(bool),
        isPrivatePullRequests: opt(bool),
        isPrivateQuestionsAnswered: opt(bool),
        isPrivateQuestionsAsked: opt(bool),
        isPrivateQuizzesCreated: opt(bool),
        isPrivateRoles: opt(bool),
        isPrivateRoutines: opt(bool),
        isPrivateRoutinesCreated: opt(bool),
        isPrivateStandards: opt(bool),
        isPrivateStandardsCreated: opt(bool),
        isPrivateTeamsCreated: opt(bool),
        isPrivateBookmarks: opt(bool),
        isPrivateVotes: opt(bool),
        profileImage: opt(imageFile),
        theme: opt(theme),
    }, [
        ["focusModes", ["Create", "Update", "Delete"], "many", "opt", focusModeValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", userTranslationValidation],
    ], [], d),
};

// Since bots are a special case of users, we must create a combined validation model 
// for the User ModelLogic object to use when creating/updating bots or updating your profile
export const userValidation: YupModel<["create", "update"]> = {
    // You can only create bots, so we can take botValidation.create directly
    create: botValidation.create,
    // For update, we must combine both botValidation.update and profileValidation.update
    update: (d) => yupObj({
        // Bot part
        id: opt(id), // Have to make this optional
        isBotDepictingPerson: opt(bool),
        botSettings: opt(botSettings),
        // Profile part
        bannerImage: opt(imageFile),
        handle: opt(handle),
        name: opt(name),
        isPrivate: opt(bool),
        isPrivateApis: opt(bool),
        isPrivateApisCreated: opt(bool),
        isPrivateCodes: opt(bool),
        isPrivateCodesCreated: opt(bool),
        isPrivateMemberships: opt(bool),
        isPrivateProjects: opt(bool),
        isPrivateProjectsCreated: opt(bool),
        isPrivatePullRequests: opt(bool),
        isPrivateQuestionsAnswered: opt(bool),
        isPrivateQuestionsAsked: opt(bool),
        isPrivateQuizzesCreated: opt(bool),
        isPrivateRoles: opt(bool),
        isPrivateRoutines: opt(bool),
        isPrivateRoutinesCreated: opt(bool),
        isPrivateStandards: opt(bool),
        isPrivateStandardsCreated: opt(bool),
        isPrivateTeamsCreated: opt(bool),
        isPrivateBookmarks: opt(bool),
        isPrivateVotes: opt(bool),
        profileImage: opt(imageFile),
        theme: opt(theme),
    }, [
        // Profile part part
        ["focusModes", ["Create", "Update", "Delete"], "many", "opt", focusModeValidation],
        // Works for both bots and profiles
        ["translations", ["Create", "Update", "Delete"], "many", "opt", userTranslationValidation],
    ], [], d),
};

/**
 * Schema for traditional email/password log in FORM.
 */
export const userDeleteOneSchema = yup.object().shape({
    password: req(yup.string().removeEmptyString().max(128, maxStrErr)),
    deletePublicData: req(bool),
});

/** Schema for sending a password reset request */
export const emailRequestPasswordChangeSchema = yup.object().shape({
    email: req(email),
});

/** Schema for resetting your password */
export const emailResetPasswordFormSchema = yup.object().shape({
    newPassword: req(password),
    confirmNewPassword: req(password).oneOf([yup.ref("newPassword")], "Passwords must match"),
});

export const emailResetPasswordSchema = yup.object().shape({
    id: req(id),
    code: req(yup.string().trim().removeEmptyString().max(128, maxStrErr)),
    newPassword: req(password),
});

export const validateSessionSchema = yup.object().shape({
    timeZone: req(yup.string().trim().removeEmptyString().max(128, maxStrErr)),
});

export const switchCurrentAccountSchema = yup.object().shape({
    id: req(id),
});

export const profileEmailUpdateValidation: YupModel<["update"]> = {
    update: (d) => yupObj({
        currentPassword: req(password),
        newPassword: opt(password),
    }, [
        ["emails", ["Create", "Delete"], "many", "opt", emailValidation],
    ], [], d),
};
