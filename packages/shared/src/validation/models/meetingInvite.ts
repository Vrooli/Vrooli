import { id, message, opt, req, YupModel, yupObj } from "../utils";

export const meetingInviteValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        message: opt(message),
    }, [
        ["meeting", ["Connect"], "one", "req"],
        ["user", ["Connect"], "many", "opt"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], d),
};
