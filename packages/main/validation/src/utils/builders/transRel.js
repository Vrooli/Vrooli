import { req } from "./req";
import { opt } from "./opt";
import { id, language } from "../commonFields";
import { yupObj } from "./yupObj";
export const transRel = (partialYupModel) => ({
    create: ({ o }) => yupObj({
        id: req(id),
        language: req(language),
        ...partialYupModel.create,
    }, [], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        language: opt(language),
        ...partialYupModel.update,
    }, [], [], o),
});
//# sourceMappingURL=transRel.js.map