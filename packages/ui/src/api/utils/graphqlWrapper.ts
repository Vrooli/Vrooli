import { ApolloError, DocumentNode } from "@apollo/client";
import { CommonKey, ErrorKey } from "@shared/translations";
import { exists } from "@shared/utils";
import { PubSub } from "../../utils/pubsub";
import { errorToCode } from "./errorParser";
import { initializeApollo } from "./initialize";

// Input type wrapped with 'input' key, as all GraphQL inputs follow this pattern. 
// If you're wondering why, it prevents us from having to define the input fields in 
// DocumentNode definitions
type InputType = { input: { [x: string]: any } }

interface BaseWrapperProps<Output extends object> {
    // Callback to determine if mutation was a success, using mutation's return data
    successCondition?: (data: Output) => boolean;
    // Message displayed on success
    successMessage?: (data: Output) => {
        key: CommonKey;
        variables?: { [key: string]: string | number };
    }
    // Callback triggered on success
    onSuccess?: (data: Output) => any;
    // Message displayed on error
    errorMessage?: (response?: ApolloError | Output) => {
        key: ErrorKey | CommonKey;
        variables?: { [key: string]: string | number };
    }
    // If true, display default error snack. Will not display if error message is set
    showDefaultErrorSnack?: boolean;
    // Callback triggered on error
    onError?: (response: ApolloError) => any;
    // Milliseconds before showing a spinner. If undefined or null, spinner disabled
    spinnerDelay?: number | null;
}

interface DocumentNodeWrapperProps<Output extends object, Input extends Record<string, any> | never> extends BaseWrapperProps<Output> {
    // DocumentNode used to create useMutation or useLazyQuery function
    node: DocumentNode;
    // data to pass into useMutation or useLazyQuery function
    input?: Input;
}

interface MutationWrapperProps<Output extends object, Input extends Record<string, any> | never> extends BaseWrapperProps<Output> {
    // useMutation function
    mutation: (options?: any) => Promise<any>;
    // data to pass into useMutation function
    input?: Input;
}

interface QueryWrapperProps<Output extends object, Input extends Record<string, any> | never> extends BaseWrapperProps<Output> {
    // useLazyQuery function
    query: (options?: any) => Promise<any>;
    // data to pass into useLazyQuery function
    input?: Input;
}

interface GraphqlWrapperHelperProps<Output extends object> extends BaseWrapperProps<Output> {
    // useMutation or useLazyQuery function
    call: () => Promise<any>;
};

/**
 * Wraps an object in "input" key, as all GraphQL inputs follow this pattern.
 * Ignores if input is undefined or null, or if input is already wrapped in "input" key
 */
const wrapInput = (input: Record<string, any> | undefined): InputType | undefined => {
    if (!exists(input)) return undefined;
    if (Object.keys(input).length === 1 && input.hasOwnProperty("input")) return input as InputType;
    return { input } as InputType;
}


/**
 * Helper function to handle response and catch of useMutation and useLazyQuery functions.
 * @param 
 */
export const graphqlWrapperHelper = <Output extends object>({
    call,
    successCondition = () => true,
    successMessage,
    onSuccess,
    errorMessage,
    showDefaultErrorSnack = true,
    onError,
    spinnerDelay = 1000
}: GraphqlWrapperHelperProps<Output>) => {
    // Helper function to handle errors
    const handleError = (data?: Output | ApolloError | undefined) => {
        // Stop spinner
        if (spinnerDelay) PubSub.get().publishLoading(false);
        // Determine if error caused by bad response, or caught error
        const isApolloError: boolean = exists(data) && data.hasOwnProperty("graphQLErrors");
        // If specific error message is set, display it
        if (typeof errorMessage === "function") {
            const { key, variables } = errorMessage(data);
            PubSub.get().publishSnack({ messageKey: key, messageVariables: variables, severity: "Error", data });
        }
        // Otherwise, if show default error snack is set, display it
        else if (showDefaultErrorSnack) {
            PubSub.get().publishSnack({ messageKey: isApolloError ? errorToCode(data as ApolloError) : "ErrorUnknown", severity: "Error", data });
        }
        // If error callback is set, call it
        if (typeof onError === "function") {
            onError((isApolloError ? data : { message: "Unknown error occurred" }) as ApolloError);
        }
    }
    // Start loading spinner
    if (spinnerDelay) PubSub.get().publishLoading(spinnerDelay);
    // Call function
    call().then((response: { data?: Output | null | undefined }) => {
        // If response is null or undefined or not an object, then there must be an error
        if (!response || !response.data || typeof response.data !== "object") {
            handleError();
            return;
        }
        // We likely need to go one layer deeper to get the actual data, since the 
        // result from Apollo is wrapped in an object with the endpoint name as the key. 
        // If this is not the case, then we are (hopefully) using a custom hook which already 
        // removes the extra layer.
        const data: Output = (Object.keys(response.data).length === 1 && !["success", "count"].includes(Object.keys(response.data)[0])) ? Object.values(response.data)[0] : response.data;
        // If this is a Count object with count = 0, then there must be an error
        if ((data as any)?.__typename === "Count" && (data as any)?.count === 0) {
            handleError(data);
            return;
        }
        if (successCondition(data)) {
            if (successMessage) PubSub.get().publishSnack({
                messageKey: successMessage(data).key,
                messageVariables: successMessage(data).variables,
                severity: "Success"
            });
            if (spinnerDelay) PubSub.get().publishLoading(false);
            if (onSuccess && typeof onSuccess === "function") onSuccess(data);
        } else {
            handleError(data);
        }
    }).catch((response: ApolloError) => {
        handleError(response);
    })
}

/**
 * Calls a useMutation or useLazyQuery function and handles response and catch, using the DocumentNode. 
 * This is useful when you want to query or mutate outside of a component (i.e. you can't create a hook)
 */
export const documentNodeWrapper = <Output extends object, Input extends Record<string, any> | never>(props: DocumentNodeWrapperProps<Output, Input>) => {
    const { node, ...rest } = props;
    // Initialize apollo client
    const client = initializeApollo();
    // Determine if DocumentNode is a mutation or query, by checking the operation of the first 
    // OperationDefinition in the definitions array
    const isMutation = node.definitions.some((def) => def.kind === "OperationDefinition" && def.operation === "mutation");
    return graphqlWrapperHelper({
        call: () => isMutation ?
            client.mutate({ mutation: node, variables: wrapInput(rest.input) }) :
            client.query({ query: node, variables: wrapInput(rest.input) }),
        ...rest
    });
}

/**
 * Wraps a useMutation function and handles response and catch
 */
export const mutationWrapper = <Output extends object, Input extends Record<string, any> | never>(props: MutationWrapperProps<Output, Input>) => {
    const { mutation, ...rest } = props;
    return graphqlWrapperHelper({
        call: () => mutation({ variables: wrapInput(rest.input) }),
        ...rest
    });
}

/**
 * Wraps a useLazyQuery function and handles response and catch
 */
export const queryWrapper = <Output extends object, Input extends Record<string, any> | never>(props: QueryWrapperProps<Output, Input>) => {
    const { query, ...rest } = props;
    return graphqlWrapperHelper({
        call: () => query({ variables: wrapInput(rest.input) }),
        ...rest
    });
}