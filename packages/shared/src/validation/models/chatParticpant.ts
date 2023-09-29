import { id, req, YupModel, yupObj } from "../utils";

export const chatParticipantValidation: YupModel<false, true> = {
    update: (d) => yupObj({
        id: req(id),
    }, [], [], d),
};
