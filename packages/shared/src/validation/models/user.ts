/* c8 ignore start */
// Coverage is ignored for validation model files because they export schema objects rather than
// executable functions. Their correctness is verified through comprehensive validation tests.
import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bio, bool, config, email, handle, id, imageFile, name, password, publicId, theme } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { PASSWORD_VERIFICATION_CODE_MAX_LENGTH, TIMEZONE_MAX_LENGTH } from "../utils/validationConstants.js";
import { botValidation } from "./bot.js";
import { emailValidation } from "./email.js";

/**
 * Schema for traditional email/password log in. NOT the form
 */
export const emailLogInSchema = yup.object().shape({
    email: opt(email),
    password: opt(yup.string().trim().removeEmptyString().max(PASSWORD_VERIFICATION_CODE_MAX_LENGTH, maxStrErr)),
    verificationCode: opt(yup.string().trim().removeEmptyString().max(PASSWORD_VERIFICATION_CODE_MAX_LENGTH, maxStrErr)),
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
        creditSettings: opt(config),
        handle: opt(handle),
        id: req(id),
        isPrivate: opt(bool),
        isPrivateMemberships: opt(bool),
        isPrivatePullRequests: opt(bool),
        isPrivateResources: opt(bool),
        isPrivateResourcesCreated: opt(bool),
        isPrivateTeamsCreated: opt(bool),
        isPrivateBookmarks: opt(bool),
        isPrivateVotes: opt(bool),
        name: opt(name),
        profileImage: opt(imageFile),
        theme: opt(theme),
    }, [
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
        isBotDepictingPerson: opt(bool),
        botSettings: opt(config),
        // Profile part
        bannerImage: opt(imageFile),
        creditSettings: opt(config),
        handle: opt(handle),
        id: req(id),
        isPrivate: opt(bool),
        isPrivateMemberships: opt(bool),
        isPrivatePullRequests: opt(bool),
        isPrivateResources: opt(bool),
        isPrivateResourcesCreated: opt(bool),
        isPrivateTeamsCreated: opt(bool),
        isPrivateBookmarks: opt(bool),
        isPrivateVotes: opt(bool),
        name: opt(name),
        profileImage: opt(imageFile),
        theme: opt(theme),
    }, [
        // Works for both bots and profiles
        ["translations", ["Create", "Update", "Delete"], "many", "opt", userTranslationValidation],
    ], [], d),
};

/**
 * Schema for traditional email/password log in FORM.
 */
export const userDeleteOneSchema = yup.object().shape({
    password: req(yup.string().removeEmptyString().max(PASSWORD_VERIFICATION_CODE_MAX_LENGTH, maxStrErr)),
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
    id: opt(id),
    publicId: opt(publicId),
    code: req(yup.string().trim().removeEmptyString().max(PASSWORD_VERIFICATION_CODE_MAX_LENGTH, maxStrErr)),
    newPassword: req(password),
}).test(
    "id-or-publicid-required",
    "Either id or publicId must be provided",
    (value) => !!value.id || !!value.publicId,
);

export const validateSessionSchema = yup.object().shape({
    timeZone: req(yup.string().trim().removeEmptyString().max(TIMEZONE_MAX_LENGTH, maxStrErr)),
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

