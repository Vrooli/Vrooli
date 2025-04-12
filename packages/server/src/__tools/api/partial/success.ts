import { Success } from "@local/shared";
import { ApiPartial } from "../types.js";

export const success: ApiPartial<Success> = {
    full: {
        success: true,
    },
};
