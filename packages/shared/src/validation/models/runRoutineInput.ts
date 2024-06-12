import * as yup from "yup";
import { id, maxStrErr, req, YupModel, yupObj } from "../utils";

const data = yup.string().trim().removeEmptyString().max(8192, maxStrErr);

export const runRoutineInputValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        data: req(data),
    }, [
        ["input", ["Connect"], "one", "req"],
        ["runRoutine", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        data: req(data),
    }, [], [], d),
};
