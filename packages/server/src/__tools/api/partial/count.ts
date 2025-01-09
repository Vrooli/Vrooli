import { Count } from "@local/shared";
import { ApiPartial } from "../types";

export const count: ApiPartial<Count> = {
    full: {
        count: true,
    },
};
