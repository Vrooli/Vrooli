import { type Success } from "@local/shared";
import { type ApiPartial } from "../types.js";

export const success: ApiPartial<Success> = {
    full: {
        success: true,
    },
};
