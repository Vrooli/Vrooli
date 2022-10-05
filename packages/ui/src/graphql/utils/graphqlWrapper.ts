import { Pubs, PubSub } from "utils";
import { ApolloCache, DefaultContext, DocumentNode, MutationFunctionOptions, QueryFunctionOptions } from '@apollo/client';
import { errorToMessage } from './errorParser';
import { ApolloError } from 'types';
import { SnackSeverity } from "components";
import { initializeApollo } from "./initialize";

// Input type wrapped with 'input' key, as all GraphQL inputs follow this pattern. 
// If you're wondering why, it prevents us from having to define the input fields in 
// DocumentNode definitions
type InputType = { input: { [x: string]: any } }

interface BaseWrapperProps<Output> {
    // Callback to determine if mutation was a success, using mutation's return data
    successCondition?: (data: Output) => boolean;
    // Message displayed on success
    successMessage?: (data: Output) => string;
    // Callback triggered on success
    onSuccess?: (data: Output) => any;
    // Message displayed on error
    errorMessage?: (response?: ApolloError | Output) => string;
    // If true, display default error snack. Will not display if error message or data is set
    showDefaultErrorSnack?: boolean;
    // Callback triggered on error
    onError?: (response: ApolloError) => any;
    // Milliseconds before showing a spinner. If undefined or null, spinner disabled
    spinnerDelay?: number | null;
}

interface DocumentNodeWrapperProps<Output, Input extends InputType> extends BaseWrapperProps<Output> {
    // DocumentNode used to create useMutation or useLazyQuery function
    node: DocumentNode;
    // data to pass into useMutation or useLazyQuery function
    input?: Input['input'];
}

interface MutationWrapperProps<Output, Input extends InputType> extends BaseWrapperProps<Output> {
    // useMutation function
    mutation: (options?: MutationFunctionOptions<Output, Input, DefaultContext, ApolloCache<any>> | undefined) => Promise<any>;
    // data to pass into useMutation function
    input?: Input['input'];
}

interface QueryWrapperProps<Output, Input extends InputType> extends BaseWrapperProps<Output> {
    // useLazyQuery function
    query: (options?: QueryFunctionOptions<Output, Input> | undefined) => Promise<any>;
    // data to pass into useLazyQuery function
    input?: Input['input'];
}

interface GraphqlWrapperHelperProps<Output> extends BaseWrapperProps<Output> {
    // useMutation or useLazyQuery function
    call: () => Promise<any>;
};

/**
 * Helper function to handle response and catch of useMutation and useLazyQuery functions.
 * @param 
 */
export const graphqlWrapperHelper = <Output>({
    call,
    successCondition = () => true,
    successMessage,
    onSuccess,
    errorMessage,
    showDefaultErrorSnack = true,
    onError,
    spinnerDelay = 1000
}: GraphqlWrapperHelperProps<Output>) => {
    if (spinnerDelay) PubSub.get().publishLoading(spinnerDelay);
    call().then((response: { data?: Output | null | undefined }) => {
        if (successCondition(response.data as Output)) {
            if (successMessage) PubSub.get().publishSnack({ message: successMessage && successMessage(response.data as Output), severity: SnackSeverity.Success });
            if (spinnerDelay) PubSub.get().publishLoading(false);
            if (onSuccess && typeof onSuccess === 'function') onSuccess(response.data as Output);
        } else {
            if (errorMessage) {
                PubSub.get().publishSnack({ message: errorMessage && errorMessage(response.data as Output), severity: SnackSeverity.Error, data: response });
            }
            else if (showDefaultErrorSnack) {
                PubSub.get().publishSnack({ message: 'Unknown error occurred.', severity: SnackSeverity.Error, data: response });
            }
            if (spinnerDelay) PubSub.get().publish(Pubs.Loading, false);
            if (onError && typeof onError === 'function') onError({ message: 'Unknown error occurred.' });
        }
    }).catch((response: ApolloError) => {
        if (spinnerDelay) PubSub.get().publishLoading(false);
        if (errorMessage) {
            PubSub.get().publishSnack({ message: errorMessage && errorMessage(response), severity: SnackSeverity.Error, data: response });
        }
        else if (showDefaultErrorSnack) {
            PubSub.get().publishSnack({ message: errorToMessage(response), severity: SnackSeverity.Error, data: response });
        }
        if (onError && typeof onError === 'function') onError(response);
    })
}

/**
 * Calls a useMutation or useLazyQuery function and handles response and catch, using the DocumentNode. 
 * This is useful when you want to query or mutate outside of a component (i.e. you can't create a hook)
 */
export const documentNodeWrapper = <Output, Input extends InputType>(props: DocumentNodeWrapperProps<Output, Input>) => {
    const { node, ...rest } = props;
    // Initialize apollo client
    const client = initializeApollo();
    // Determine if DocumentNode is a mutation or query, by checking the operation of the first 
    // OperationDefinition in the definitions array
    const isMutation = node.definitions.some((def) => def.kind === 'OperationDefinition' && def.operation === 'mutation');
    return graphqlWrapperHelper({
        call: () => isMutation ?
            client.mutate({ mutation: node, variables: rest.input ? { input: rest.input } as Input : undefined }) :
            client.query({ query: node, variables: rest.input ? { input: rest.input } as Input : undefined }),
        ...rest
    });
}

/**
 * Wraps a useMutation function and handles response and catch
 */
export const mutationWrapper = <Output, Input extends InputType>(props: MutationWrapperProps<Output, Input>) => {
    const { mutation, ...rest } = props;
    return graphqlWrapperHelper({
        call: () => mutation({ variables: rest.input ? { input: rest.input } as Input : undefined }),
        ...rest
    });
}

/**
 * Wraps a useLazyQuery function and handles response and catch
 */
export const queryWrapper = <Output, Input extends InputType>(props: QueryWrapperProps<Output, Input>) => {
    const { query, ...rest } = props;
    return graphqlWrapperHelper({
        call: () => query({ variables: rest.input ? { input: rest.input } as Input : undefined }),
        ...rest
    });
}