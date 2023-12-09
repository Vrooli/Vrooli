import { PullRequestStatus, PullRequestToObjectType } from "@local/shared";
import * as yup from "yup";
import { enumToYup, id, maxStrErr, minStrErr, opt, req, transRel, YupModel, yupObj } from "../utils";

const pullRequestTo = enumToYup(PullRequestToObjectType);
const pullRequestStatus = enumToYup(PullRequestStatus);
const text = yup.string().trim().removeEmptyString().min(1, minStrErr).max(32768, maxStrErr);

export const pullRequestTranslationValidation: YupModel = transRel({
    create: () => ({
        text: req(text),
    }),
    update: () => ({
        text: opt(text),
    }),
});

export const pullRequestValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        toObjectType: req(pullRequestTo),
    }, [
        ["to", ["Connect"], "one", "req"],
        ["from", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", pullRequestTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        status: opt(pullRequestStatus),
    }, [
        ["translations", ["Delete", "Create", "Update"], "many", "opt", pullRequestTranslationValidation],
    ], [], d),
};
