import { id, req, YupModel, yupObj } from "../utils";

export const chatParticipantValidation: YupModel<false, true> = {
    update: ({ o }) => yupObj({
        id: req(id),
    }, [], [], o),
};
