import { useEffect } from "react";
import { errorToMessage } from "../../api";
import { PubSub } from "../pubsub";
export const useDisplayApolloError = (error) => {
    useEffect(() => {
        if (error) {
            const message = errorToMessage(error);
            PubSub.get().publishSnack({ message, severity: "Error" });
        }
    }, [error]);
};
//# sourceMappingURL=useDisplayApolloError.js.map