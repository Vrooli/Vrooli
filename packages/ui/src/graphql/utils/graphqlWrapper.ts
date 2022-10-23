import { PubSub } from "utils";
import { DocumentNode } from '@apollo/client';
import { errorToMessage } from './errorParser';
import { ApolloError } from 'types';
import { SnackSeverity } from "components";
import { initializeApollo } from "./initialize";

// Input type wrapped with 'input' key, as all GraphQL inputs follow this pattern. 
// If you're wondering why, it prevents us from having to define the input fields in 
// DocumentNode definitions
type InputType = { input: { [x: string]: any } }

interface BaseWrapperProps<Output extends object> {
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

interface DocumentNodeWrapperProps<Output extends object, Input extends InputType> extends BaseWrapperProps<Output> {
    // DocumentNode used to create useMutation or useLazyQuery function
    node: DocumentNode;
    // data to pass into useMutation or useLazyQuery function
    input?: Input['input'];
}

interface MutationWrapperProps<Output extends object, Input extends InputType> extends BaseWrapperProps<Output> {
    // useMutation function
    mutation: (options?: any) => Promise<any>;
    // data to pass into useMutation function
    input?: Input['input'];
}

interface QueryWrapperProps<Output extends object, Input extends InputType> extends BaseWrapperProps<Output> {
    // useLazyQuery function
    query: (options?: any) => Promise<any>;
    // data to pass into useLazyQuery function
    input?: Input['input'];
}

interface GraphqlWrapperHelperProps<Output extends object> extends BaseWrapperProps<Output> {
    // useMutation or useLazyQuery function
    call: () => Promise<any>;
};

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
        const isApolloError: boolean = data !== undefined && data !== null && data.hasOwnProperty('graphQLErrors');
        // Determine message to display, if any
        const message: string | undefined = typeof errorMessage === 'function' ? 
            errorMessage(data) :
            showDefaultErrorSnack ?
                isApolloError ?
                    errorToMessage(data as ApolloError) :
                    errorToMessage({ message: 'Unknown error occurred' }) :
                undefined;
        if (message) {
            PubSub.get().publishSnack({ message, severity: SnackSeverity.Error, data });
        }
        // Determine if error callback should be called
        if (typeof onError === 'function') {
            onError(isApolloError ? data as ApolloError : { message: 'Unknown error occurred' });
        }
    }
    // Start loading spinner
    if (spinnerDelay) PubSub.get().publishLoading(spinnerDelay);
    // Call function
    call().then((response: { data?: Output | null | undefined }) => {
        // We need to go one layer deeper to get the actual data. 
        // If this doesn't exist, then there must be an error
        if (!response.data || 
            typeof response.data !== 'object' || 
            Array.isArray(response.data) || 
            Object.keys(response.data).length === 0) {
            handleError();
            return;
        }
        // Get object/primitive inside response.data
        const data: Output = Object.values(response.data)[0];
        // If this is a Count object with count = 0, then there must be an error
        if ((data as any)?.__typename === 'Count' && (data as any)?.count === 0) {
            handleError(data);
            return;
        }
        if (successCondition(data)) {
            if (successMessage) PubSub.get().publishSnack({ message: successMessage && successMessage(data), severity: SnackSeverity.Success });
            if (spinnerDelay) PubSub.get().publishLoading(false);
            if (onSuccess && typeof onSuccess === 'function') onSuccess(data);
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
export const documentNodeWrapper = <Output extends object, Input extends InputType>(props: DocumentNodeWrapperProps<Output, Input>) => {
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
export const mutationWrapper = <Output extends object, Input extends InputType>(props: MutationWrapperProps<Output, Input>) => {
    const { mutation, ...rest } = props;
    return graphqlWrapperHelper({
        call: () => mutation({ variables: rest.input ? { input: rest.input } as Input : undefined }),
        ...rest
    });
}

/**
 * Wraps a useLazyQuery function and handles response and catch
 */
export const queryWrapper = <Output extends object, Input extends InputType>(props: QueryWrapperProps<Output, Input>) => {
    const { query, ...rest } = props;
    return graphqlWrapperHelper({
        call: () => query({ variables: rest.input ? { input: rest.input } as Input : undefined }),
        ...rest
    });
}