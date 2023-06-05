import { errorToMessage, ServerResponse } from "api";
import { useEffect } from "react";
import { PubSub } from "utils/pubsub";

/**
 * When the server throws an error, this function will display
 * it as a snack bar notification
 * @param errors Errors from the server
 */
export const useDisplayServerError = (errors: ServerResponse["errors"]) => {
    useEffect(() => {
        if (errors) {
            for (const error of errors) {
                const message = errorToMessage({ errors: [error] });
                PubSub.get().publishSnack({ message, severity: "Error" });
            }
        }
    }, [errors]);
};
