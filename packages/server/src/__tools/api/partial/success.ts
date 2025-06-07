import { type Success } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const success: ApiPartial<Success> = {
    full: {
        success: true,
    },
};
