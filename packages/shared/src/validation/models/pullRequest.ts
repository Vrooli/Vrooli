import * as yup from "yup";
import { PullRequestStatus, PullRequestToObjectType } from "../../api/generated/graphqlTypes";
import { YupModel, enumToYup, id, maxStrErr, minStrErr, opt, req, transRel, yupObj } from "../utils";

const pullRequestTo = enumToYup(PullRequestToObjectType);
const pullRequestStatus = enumToYup(PullRequestStatus);
const text = yup.string().trim().removeEmptyString().min(1, minStrErr).max(32768, maxStrErr);

export const pullRequestTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        text: req(text),
    }),
    update: () => ({
        text: opt(text),
    }),
});

export const pullRequestValidation: YupModel<["create", "update"]> = {
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
