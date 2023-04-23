import { ApolloError } from "@apollo/client";
import { useEffect } from "react";
import { errorToMessage } from "../../api";
import { PubSub } from "../pubsub";

/**
 * When an Apollo query or mutation throws an error, this function will display
 * it as a snack bar notification
 * @param error The error to display
 */
export const useDisplayApolloError = (error: ApolloError | undefined) => {
    useEffect(() => {
        if (error) {
            const message = errorToMessage(error);
            PubSub.get().publishSnack({ message, severity: "Error" });
        }
    }, [error]);
};
