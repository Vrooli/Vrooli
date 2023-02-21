import * as yup from 'yup';
import { PullRequestStatus, PullRequestToObjectType } from '@shared/consts';
import { blankToUndefined, enumToYup, id, maxStrErr, minStrErr, opt, req, transRel, YupModel, yupObj } from "../utils";

const pullRequestTo = enumToYup(PullRequestToObjectType);
const pullRequestStatus = enumToYup(PullRequestStatus);
const text = yup.string().transform(blankToUndefined).min(1, minStrErr).max(32768, maxStrErr)

export const pullRequestTranslationValidation: YupModel = transRel({
    create: {
        text: req(text),
    },
    update: {
        text: opt(text),
    }
})

export const pullRequestValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        toObjectType: req(pullRequestTo),
    }, [
        ['to', ['Connect'], 'one', 'req'],
        ['from', ['Connect'], 'one', 'req'],
        ['translations', ['Create'], 'many', 'opt', pullRequestTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        status: opt(pullRequestStatus),
    }, [
        ['translations', ['Delete', 'Create', 'Update'], 'many', 'opt', pullRequestTranslationValidation],
    ], [], o),
}