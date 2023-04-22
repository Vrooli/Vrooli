import { id, message, opt, req, YupModel, yupObj } from "../utils";

export const meetingInviteValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        message: opt(message),
    }, [
        ["meeting", ["Connect"], "one", "req"],
        ["user", ["Connect"], "many", "opt"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], o),
};
