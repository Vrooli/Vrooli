import { useState } from "react";

type LazyFetchState = {
    data: any;
    errors: any;
    loading: boolean;
};
const defaultLazyFetchState: LazyFetchState = { data: null, errors: null, loading: false };

// Variables to hold the mock functions and state
export let mockLazyFetchData = jest.fn();
export let setMockLazyFetchState: React.Dispatch<React.SetStateAction<LazyFetchState>> | null = null;

// Function to reset the mocks before each test
export function resetLazyFetchMocks() {
    mockLazyFetchData = jest.fn();
    setMockLazyFetchState = null;
}

// The mocked useLazyFetch hook
export function useLazyFetch<TInput extends Record<string, any> | undefined, TData>() {
    const [fetchResult, setFetchResult] = useState<LazyFetchState>(defaultLazyFetchState);
    setMockLazyFetchState = setFetchResult;
    return [mockLazyFetchData, fetchResult];
}
