import { id, req, YupModel, yupObj } from "../utils";

export const chatParticipantValidation: YupModel<["update"]> = {
    update: (d) => yupObj({
        id: req(id),
    }, [], [], d),
};
