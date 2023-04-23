import * as yup from "yup";
import { bio, blankToUndefined, bool, email, handle, maxStrErr, name, opt, password, req, theme, transRel, yupObj } from "../utils";
import { emailValidation } from "./email";
import { focusModeValidation } from "./focusMode";
export const emailLogInSchema = yup.object().shape({
    email: opt(email),
    password: opt(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
    verificationCode: opt(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
});
export const userTranslationValidation = transRel({
    create: {
        bio: opt(bio),
    },
    update: {
        bio: opt(bio),
    },
});
export const userValidation = {
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
export const userDeleteOneSchema = yup.object().shape({
    password: req(yup.string().transform(blankToUndefined).max(128, maxStrErr)),
    deletePublicData: req(bool),
});
export const emailRequestPasswordChangeSchema = yup.object().shape({
    email: req(email),
});
export const emailResetPasswordSchema = yup.object().shape({
    newPassword: req(password),
    confirmNewPassword: yup.string().oneOf([yup.ref("newPassword"), null], "Passwords must match"),
});
export const profileEmailUpdateValidation = {
    update: ({ o }) => yupObj({
        currentPassword: req(password),
        newPassword: opt(password),
    }, [
        ["emails", ["Create", "Delete"], "many", "opt", emailValidation],
    ], [], o),
};
//# sourceMappingURL=user.js.map