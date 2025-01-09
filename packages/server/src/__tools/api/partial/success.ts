import { Success } from "@local/shared";
import { ApiPartial } from "../types";

export const success: ApiPartial<Success> = {
    full: {
        success: true,
    },
};
