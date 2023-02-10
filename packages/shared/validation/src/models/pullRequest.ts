import { PullRequestStatus, PullRequestToObjectType } from '@shared/consts';
import { enumToYup, id, opt, req, YupModel, yupObj } from "../utils";

const pullRequestTo = enumToYup(PullRequestToObjectType);

const pullRequestStatus = enumToYup(PullRequestStatus);

export const pullRequestValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        toObjectType: req(pullRequestTo),
    }, [
        ['to', ['Connect'], 'one', 'req'],
        ['from', ['Connect'], 'one', 'req'],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        status: opt(pullRequestStatus),
    }, [], [], o),
}