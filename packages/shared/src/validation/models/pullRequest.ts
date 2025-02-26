import * as yup from "yup";
import { PullRequestStatus, PullRequestToObjectType } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { maxStrErr, minStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";

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
