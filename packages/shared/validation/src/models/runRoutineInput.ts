import * as yup from "yup";
import { blankToUndefined, id, maxStrErr, req, YupModel, yupObj } from "../utils";

const data = yup.string().transform(blankToUndefined).max(8192, maxStrErr);

export const runRoutineInputValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        data: req(data),
    }, [
        ["input", ["Connect"], "one", "req"],
        ["runRoutine", ["Connect"], "one", "req"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        data: req(data),
    }, [], [], o),
};
