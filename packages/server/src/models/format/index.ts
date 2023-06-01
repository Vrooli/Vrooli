import { GqlModelType } from "@local/shared";
import { Formatter } from "../types";
import { ApiFormat } from "./api";

/**
 * Maps model types to their respective formatter logic.
 */
export const FormatMap: { [key in GqlModelType]?: Formatter<any> } = {
    Api: ApiFormat,
};
