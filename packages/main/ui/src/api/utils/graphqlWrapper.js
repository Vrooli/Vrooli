import { exists } from "@local/utils";
import { PubSub } from "../../utils/pubsub";
import { errorToCode } from "./errorParser";
import { initializeApollo } from "./initialize";
;
const wrapInput = (input) => {
    if (!exists(input))
        return undefined;
    if (Object.keys(input).length === 1 && input.hasOwnProperty("input"))
        return input;
    return { input };
};
export const graphqlWrapperHelper = ({ call, successCondition = () => true, successMessage, onSuccess, errorMessage, showDefaultErrorSnack = true, onError, spinnerDelay = 1000, }) => {
    const handleError = (data) => {
        if (spinnerDelay)
            PubSub.get().publishLoading(false);
        const isApolloError = exists(data) && data.hasOwnProperty("graphQLErrors");
        if (typeof errorMessage === "function") {
            const { key, variables } = errorMessage(data);
            PubSub.get().publishSnack({ messageKey: key, messageVariables: variables, severity: "Error", data });
        }
        else if (showDefaultErrorSnack) {
            PubSub.get().publishSnack({ messageKey: isApolloError ? errorToCode(data) : "ErrorUnknown", severity: "Error", data });
        }
        if (typeof onError === "function") {
            onError((isApolloError ? data : { message: "Unknown error occurred" }));
        }
    };
    if (spinnerDelay)
        PubSub.get().publishLoading(spinnerDelay);
    call().then((response) => {
        if (!response || !response.data || typeof response.data !== "object") {
            handleError();
            return;
        }
        const data = (Object.keys(response.data).length === 1 && !["success", "count"].includes(Object.keys(response.data)[0])) ? Object.values(response.data)[0] : response.data;
        if (data?.__typename === "Count" && data?.count === 0) {
            handleError(data);
            return;
        }
        if (successCondition(data)) {
            if (successMessage)
                PubSub.get().publishSnack({
                    messageKey: successMessage(data).key,
                    messageVariables: successMessage(data).variables,
                    severity: "Success",
                });
            if (spinnerDelay)
                PubSub.get().publishLoading(false);
            if (onSuccess && typeof onSuccess === "function")
                onSuccess(data);
        }
        else {
            handleError(data);
        }
    }).catch((response) => {
        handleError(response);
    });
};
export const documentNodeWrapper = (props) => {
    const { node, ...rest } = props;
    const client = initializeApollo();
    const isMutation = node.definitions.some((def) => def.kind === "OperationDefinition" && def.operation === "mutation");
    return graphqlWrapperHelper({
        call: () => isMutation ?
            client.mutate({ mutation: node, variables: wrapInput(rest.input) }) :
            client.query({ query: node, variables: wrapInput(rest.input) }),
        ...rest,
    });
};
export const mutationWrapper = (props) => {
    const { mutation, ...rest } = props;
    return graphqlWrapperHelper({
        call: () => mutation({ variables: wrapInput(rest.input) }),
        ...rest,
    });
};
export const queryWrapper = (props) => {
    const { query, ...rest } = props;
    return graphqlWrapperHelper({
        call: () => query({ variables: wrapInput(rest.input) }),
        ...rest,
    });
};
//# sourceMappingURL=graphqlWrapper.js.map