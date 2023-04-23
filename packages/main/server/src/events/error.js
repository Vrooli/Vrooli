import { ApolloError } from "apollo-server-express";
import i18next from "i18next";
import { randomString } from "../auth";
import { logger } from "./logger";
function genTrace(locationCode) {
    return `${locationCode}-${randomString(4)}`;
}
export class CustomError extends ApolloError {
    constructor(traceBase, errorCode, languages, data) {
        const lng = languages.length > 0 ? languages[0] : "en";
        const message = i18next.t(`error:${errorCode}`, { lng }) ?? errorCode;
        const trace = genTrace(traceBase);
        const displayMessage = (message.endsWith(".") ? message.slice(0, -1) : message) + `: ${trace}`;
        super(displayMessage, errorCode);
        Object.defineProperty(this, "name", { value: errorCode });
        if (trace) {
            logger.error({ ...(data ?? {}), msg: message, trace });
        }
    }
}
//# sourceMappingURL=error.js.map