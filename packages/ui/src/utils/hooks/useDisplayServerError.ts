import { ApolloError } from "@apollo/client";
import { errorToMessage } from "api";
import { useEffect } from "react";
import { PubSub } from "utils/pubsub";

/**
 * When the server throws an error, this function will display
 * it as a snack bar notification
 * @param error The error to display
 */
export const useDisplayServerError = (error: ApolloError | undefined) => {
    useEffect(() => {
        if (error) {
            const message = errorToMessage(error);
            PubSub.get().publishSnack({ message, severity: "Error" });
        }
    }, [error]);
};
