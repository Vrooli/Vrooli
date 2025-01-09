import { DUMMY_ID, ModelType, base36ToUuid } from "@local/shared";
import { renderHook } from "@testing-library/react";
import { act } from "react";
import { PubSub as PubSubMock } from "../utils/__mocks__/pubsub";
import { defaultYou } from "../utils/display/listTools";
import { fullPreferences, removeCookiePartialData, setCookie, setCookieFormData, setCookiePartialData } from "../utils/localStorage";
import { PubSub } from "../utils/pubsub";
import { mockLazyFetchData, useLazyFetch as mockUseLazyFetch, resetLazyFetchMocks, setMockLazyFetchState } from "./__mocks__/useLazyFetch";
import { useLazyFetch } from "./useLazyFetch";
import { applyDataTransform, fetchDataUsingUrl, getCachedData, getStoredFormData, handleInvalidUrlParams, initializeObjectState, promptToUseStoredFormData, shouldFetchData, useManagedObject } from "./useManagedObject";

jest.mock("./useLazyFetch");
jest.mock("route");

describe("useManagedObject Hook", () => {
    function mockTransform(data) {
        return { ...data, transformed: true };
    }
    const mockOverrideObject = { id: "override-id", name: "Override Object" };
    const urlBase36 = "2y5rvjtvc6yaeo4f1vvtsxbvp";
    const urlId = base36ToUuid(urlBase36);
    const mockTransformedObject = { ...mockOverrideObject, transformed: true };
    let originalLocation: Location;
    let originalUseLazyFetch: any;
    let originalPubSubMethods;

    beforeAll(() => {
        // Save the original window.location
        originalLocation = window.location;
    });
    beforeEach(() => {
        jest.clearAllMocks();
        resetLazyFetchMocks();
        global.localStorage.clear();
        setCookie("Preferences", fullPreferences);
        // Mock window.location
        delete (window as any).location;
        const origin = "http://localhost";
        const pathname = `/chat/${urlBase36}`;
        (window as any).location = {
            pathname,
            origin,
            href: `${origin}${pathname}`,
        };
        // Mock useLazyFetch
        originalUseLazyFetch = { ...useLazyFetch };
        Object.assign(useLazyFetch, mockUseLazyFetch);
        // Mock PubSub
        originalPubSubMethods = { ...PubSub };
        Object.assign(PubSub, PubSubMock);
        PubSubMock.resetMock();
    });
    afterEach(() => {
        // Restore useLazyFetch
        Object.assign(useLazyFetch, originalUseLazyFetch);
        // Restore PubSub
        Object.assign(PubSub, originalPubSubMethods);
    });
    afterAll(() => {
        jest.restoreAllMocks();
        // Restore the original window.location
        window.location = originalLocation;
    });

    it("uses overrideObject when provided and does not fetch data", () => {
        const initialFetchResult = { data: null, errors: null, loading: false };
        act(() => {
            setMockLazyFetchState?.(initialFetchResult);
        });

        const { result } = renderHook(() =>
            useManagedObject({
                endpoint: "/test-endpoint",
                objectType: "Chat",
                overrideObject: mockOverrideObject,
                transform: mockTransform,
            }),
        );

        expect(mockLazyFetchData).not.toHaveBeenCalled();
        expect(result.current.object).toEqual(mockTransformedObject);
        expect(result.current.isLoading).toBe(false);
        // We don't need to check for every permission, just that it looks good enough
        expect(result.current.permissions).toEqual(expect.objectContaining({ canDelete: expect.any(Boolean) }));
    });

    it("does not fetch data when disabled is true and returns empty object", () => {
        const initialFetchResult = { data: null, errors: null, loading: false };
        act(() => {
            setMockLazyFetchState?.(initialFetchResult);
        });

        const { result } = renderHook(() =>
            useManagedObject({
                endpoint: "/test-endpoint",
                objectType: "Chat",
                disabled: true,
                transform: mockTransform,
            }),
        );

        expect(mockLazyFetchData).not.toHaveBeenCalled();
        expect(result.current.object).toEqual(mockTransform({}));
        expect(result.current.isLoading).toBe(false);
        expect(result.current.permissions).toEqual(defaultYou);
    });

    it("uses cached data when available", () => {
        const initialFetchResult = { data: null, errors: null, loading: true };
        act(() => {
            setMockLazyFetchState?.(initialFetchResult);
        });


        // Store partial data
        const mockCachedData = { __typename: "Chat", id: urlId, name: "Cached Object" };
        setCookiePartialData(mockCachedData as any, "full");

        console.log("yeet before");
        const { result, rerender } = renderHook(() =>
            useManagedObject({
                endpoint: "/test-endpoint",
                objectType: "Chat",
                transform: mockTransform,
            }),
        );
        console.log("yeet intermediate");

        expect(result.current.object).toEqual(mockTransform(mockCachedData));
        expect(result.current.isLoading).toBe(true);

        const mockFetchedData = { __typename: "Chat", id: urlId, name: "Fetched Object" };
        act(() => {
            setMockLazyFetchState?.({ data: mockFetchedData, errors: null, loading: false });
            rerender();
        });

        console.log("yeet after");

        expect(result.current.object).toEqual(mockTransform(mockFetchedData));
        expect(result.current.isLoading).toBe(false);
        expect(setCookiePartialData).toHaveBeenCalledWith(mockFetchedData, "full");
    });

    it("uses stored form data when isCreate is true", () => {
        const initialFetchResult = { data: null, errors: null, loading: false };
        act(() => {
            setMockLazyFetchState?.(initialFetchResult);
        });

        // Store form data
        const mockFormData = { __typename: "Chat", id: urlId, name: "Form Data" };
        setCookieFormData("Chat", urlId, mockFormData);

        const { result } = renderHook(() =>
            useManagedObject({
                endpoint: "/test-endpoint",
                objectType: "Chat",
                isCreate: true,
                transform: mockTransform,
            }),
        );

        expect(result.current.object).toEqual(mockTransform(mockFormData));
        expect(result.current.isLoading).toBe(false);
    });

    it("fetches data when shouldFetchData returns true", async () => {
        const initialFetchResult = { data: null, errors: null, loading: true };
        act(() => {
            setMockLazyFetchState?.(initialFetchResult);
        });

        const { result, rerender } = renderHook(() =>
            useManagedObject({
                endpoint: "/test-endpoint",
                objectType: "Chat",
            }),
        );

        expect(mockLazyFetchData).toHaveBeenCalledWith({ id: urlId }, expect.any(Object));
        expect(result.current.isLoading).toBe(true);

        const mockFetchedData = { __typename: "TestObject", id: urlId, name: "Fetched Object" };
        act(() => {
            setMockLazyFetchState?.({ data: mockFetchedData, errors: null, loading: false });
            rerender();
        });

        expect(result.current.object).toEqual(mockFetchedData);
        expect(result.current.isLoading).toBe(false);
        expect(setCookiePartialData).toHaveBeenCalledWith(mockFetchedData, "full");
    });

    it("handles errors during fetching and calls onError if provided", async () => {
        const initialFetchResult = { data: null, errors: [{ message: "Error" }], loading: false };
        act(() => {
            setMockLazyFetchState?.(initialFetchResult);
        });

        const mockOnError = jest.fn();

        const { result } = renderHook(() =>
            useManagedObject({
                endpoint: "/test-endpoint",
                objectType: "Chat",
                onError: mockOnError,
            }),
        );

        expect(mockLazyFetchData).toHaveBeenCalled();
        expect(mockOnError).toHaveBeenCalledWith([{ message: "Error" }]);
        expect(result.current.object).toEqual({});
    });

    it("calls onInvalidUrlParams when URL params are invalid", () => {
        (window as any).location = { pathname: "/test-path" }; // URL has no object information

        const initialFetchResult = { data: null, errors: null, loading: false };
        act(() => {
            setMockLazyFetchState?.(initialFetchResult);
        });

        const mockOnInvalidUrlParams = jest.fn();

        renderHook(() =>
            useManagedObject({
                endpoint: "/test-endpoint",
                objectType: "Chat",
                onInvalidUrlParams: mockOnInvalidUrlParams,
            }),
        );

        expect(mockLazyFetchData).not.toHaveBeenCalled();
        expect(mockOnInvalidUrlParams).toHaveBeenCalledWith({});
    });

    it("updates permissions when object changes", () => {
        const initialFetchResult = { data: null, errors: null, loading: true };
        act(() => {
            setMockLazyFetchState?.(initialFetchResult);
        });

        const { result, rerender } = renderHook(() =>
            useManagedObject({
                endpoint: "/test-endpoint",
                objectType: "Chat",
                transform: mockTransform,
            }),
        );

        // We don't need to check for every permission, just that it looks good enough
        expect(result.current.permissions).toEqual(expect.objectContaining({ canDelete: expect.any(Boolean) }));

        const mockFetchedData = { __typename: "TestObject", id: urlId, name: "Fetched Object" };
        act(() => {
            setMockLazyFetchState?.({ data: mockFetchedData, errors: null, loading: false });
            rerender();
        });

        // We don't need to check for every permission, just that it looks good enough
        expect(result.current.permissions).toEqual(expect.objectContaining({ canDelete: expect.any(Boolean) }));
    });

    it("does not fetch data when overrideObject is provided even if shouldFetchData is true", () => {
        const initialFetchResult = { data: null, errors: null, loading: true };
        act(() => {
            setMockLazyFetchState?.(initialFetchResult);
        });

        renderHook(() =>
            useManagedObject({
                endpoint: "/test-endpoint",
                objectType: "Chat",
                overrideObject: mockOverrideObject,
            }),
        );

        expect(mockLazyFetchData).not.toHaveBeenCalled();
    });

    it("prompts to use stored form data when there is a conflict", () => {
        const initialFetchResult = { data: null, errors: null, loading: false };
        act(() => {
            setMockLazyFetchState?.(initialFetchResult);
        });

        const mockFormData = { id: urlId, name: "Form Data" };
        const mockCachedData = { __typename: "Chat", id: urlId, name: "Cached Data" };

        setCookieFormData("Chat", urlId, mockFormData);
        setCookiePartialData(mockCachedData as any, "full");

        const mockSubscriber = jest.fn();
        PubSub.get().subscribe("snack", mockSubscriber);

        renderHook(() =>
            useManagedObject({
                endpoint: "/test-endpoint",
                objectType: "Chat",
                isCreate: false,
                transform: mockTransform,
            }),
        );

        expect(mockSubscriber).toHaveBeenCalledWith(expect.any(Object));
    });
});

describe("Helper Functions", () => {
    describe("applyDataTransform", () => {
        it("applies the transform function to data", () => {
            const data = { name: "Test" };
            const transform = jest.fn((data) => ({ ...data, transformed: true }));
            const result = applyDataTransform(data as object, transform);
            expect(transform).toHaveBeenCalledWith(data);
            expect(result).toEqual({ name: "Test", transformed: true });
        });

        it("returns data as is when transform is not provided", () => {
            const data = { name: "Test" };
            const result = applyDataTransform(data as object);
            expect(result).toEqual(data);
        });

        it("returns empty object when data is null", () => {
            const result = applyDataTransform(null);
            expect(result).toEqual({});
        });
    });

    describe("fetchDataUsingUrl", () => {
        let mockFetchData: jest.Mock;

        beforeEach(() => {
            mockFetchData = jest.fn();
        });

        it("fetches data using handle", () => {
            const params = { handle: "test-handle" };
            const result = fetchDataUsingUrl(params, mockFetchData);
            expect(mockFetchData).toHaveBeenCalledWith({ handle: "test-handle" }, expect.any(Object));
            expect(result).toBe(true);
        });

        it("fetches data using id when handle is not available", () => {
            const params = { id: "12345" };
            const result = fetchDataUsingUrl(params, mockFetchData);
            expect(mockFetchData).toHaveBeenCalledWith({ id: "12345" }, expect.any(Object));
            expect(result).toBe(true);
        });

        it("returns false when no valid params are available", () => {
            const params = {};
            const result = fetchDataUsingUrl(params, mockFetchData);
            expect(mockFetchData).not.toHaveBeenCalled();
            expect(result).toBe(false);
        });
    });

    describe("initializeObjectState", () => {
        function mockTransform(data) {
            return { ...data, transformed: true };
        }
        const mockUrlParams = { id: "12345" };

        beforeEach(() => {
            jest.clearAllMocks();
            global.localStorage.clear();
            setCookie("Preferences", fullPreferences);
        });

        afterAll(() => {
            global.localStorage.clear();
            jest.restoreAllMocks();
        });

        it("uses overrideObject when provided", () => {
            const result = initializeObjectState({
                disabled: false,
                isCreate: false,
                objectType: "TestObject" as ModelType,
                overrideObject: { id: "override-id", isOverride: true } as any,
                transform: mockTransform,
                urlParams: mockUrlParams,
            });

            expect(result).toEqual({
                // Override object with transform applied
                initialObject: { id: "override-id", isOverride: true, transformed: true },
                hasStoredFormDataConflict: false,
            });
        });

        it("returns empty object when disabled is true", () => {
            const result = initializeObjectState({
                disabled: true,
                isCreate: false,
                objectType: "TestObject" as ModelType,
                transform: mockTransform,
                urlParams: mockUrlParams,
            });

            expect(result).toEqual({
                // Returned object is empty object with transform applied
                initialObject: { transformed: true },
                hasStoredFormDataConflict: false,
            });
        });

        it("uses cached data when available", () => {
            // Store partial data
            setCookiePartialData({ __typename: "TestObject" as ModelType, id: mockUrlParams.id, isPartial: true } as any, "full");
            // Also store form data to ensure it's not used
            setCookieFormData("TestObject" as ModelType, DUMMY_ID, { id: mockUrlParams.id, isPartial: false });

            const result = initializeObjectState({
                disabled: false,
                isCreate: false,
                objectType: "TestObject" as ModelType,
                transform: mockTransform,
                urlParams: mockUrlParams,
            });

            expect(result).toEqual({
                // Returned object is partial data with transform applied
                initialObject: { __typename: "TestObject", id: mockUrlParams.id, isPartial: true, transformed: true },
                hasStoredFormDataConflict: false,
            });
        });

        it("uses form data when isCreate is true", () => {
            // Store form data
            setCookieFormData("TestObject" as ModelType, DUMMY_ID, { id: "form-id", isForm: true });
            // Also store partial data to ensure it's not used
            setCookiePartialData({ __typename: "TestObject" as ModelType, id: "form-id", isForm: false } as any, "full");

            const result = initializeObjectState({
                disabled: false,
                isCreate: true,
                objectType: "TestObject" as ModelType,
                transform: mockTransform,
                urlParams: mockUrlParams,
            });

            expect(result).toEqual({
                // Returned object is form data with transform applied
                initialObject: { id: "form-id", isForm: true, transformed: true },
                hasStoredFormDataConflict: false,
            });
        });
    });

    describe("shouldFetchData", () => {
        it("returns true when not disabled and no overrideObject", () => {
            expect(shouldFetchData({ disabled: false })).toBe(true);
        });

        it("returns false when disabled", () => {
            expect(shouldFetchData({ disabled: true })).toBe(false);
        });

        it("returns false when overrideObject is provided", () => {
            expect(shouldFetchData({ disabled: false, overrideObject: {} })).toBe(false);
        });
    });

    describe("handleInvalidUrlParams", () => {
        let originalPubSubMethods;
        beforeEach(() => {
            originalPubSubMethods = { ...PubSub };
            Object.assign(PubSub, PubSubMock);
            PubSubMock.resetMock();
            jest.clearAllMocks();
        });

        afterEach(() => {
            Object.assign(PubSub, originalPubSubMethods);
        });

        it("returns when transform is provided", () => {
            const result = handleInvalidUrlParams({
                transform: jest.fn(),
                onInvalidUrlParams: jest.fn(),
                urlParams: {},
            });

            expect(result).toBeUndefined();
        });

        it("calls onInvalidUrlParams when provided", () => {
            const onInvalidUrlParams = jest.fn();
            handleInvalidUrlParams({
                onInvalidUrlParams,
                urlParams: { id: "invalid" },
            });

            expect(onInvalidUrlParams).toHaveBeenCalledWith({ id: "invalid" });
        });

        it("publishes default error when no handlers are provided", () => {
            const mockSubscriber = jest.fn();
            PubSub.get().subscribe("snack", mockSubscriber);

            handleInvalidUrlParams({
                urlParams: {},
            });

            expect(mockSubscriber).toHaveBeenCalledWith(expect.any(Object));
        });
    });

    describe("getCachedData", () => {
        beforeEach(() => {
            jest.clearAllMocks();
            global.localStorage.clear();
            setCookie("Preferences", fullPreferences);
        });

        afterAll(() => {
            global.localStorage.clear();
            jest.restoreAllMocks();
        });

        it("returns cached data when available", () => {
            const objectType = "TestObject" as ModelType;
            const urlParams = { id: "12345" };
            const cachedData = { __typename: objectType, id: urlParams.id, name: "Cached Object" };

            // Store the cached data
            setCookiePartialData(cachedData, "full");

            const result = getCachedData({ objectType, urlParams });
            expect(result).toEqual(cachedData);
        });

        it("returns the data we passed in (i.e. all we know about the object) when no cached data is available", () => {
            const objectType = "TestObject" as ModelType;
            const urlParams = { id: "12345" };

            // Ensure no data is cached
            removeCookiePartialData({ __typename: objectType, ...urlParams });

            const result = getCachedData({ objectType, urlParams });
            expect(result).toEqual({ __typename: objectType, ...urlParams });
        });
    });

    describe("getStoredFormData", () => {
        beforeEach(() => {
            jest.clearAllMocks();
            global.localStorage.clear();
            setCookie("Preferences", fullPreferences);
        });
        afterAll(() => {
            global.localStorage.clear();
            jest.restoreAllMocks();
        });


        describe("returns stored form data when available", () => {
            test("create forms", () => {
                const objectType = "TestObject" as ModelType;
                const objectId = "12345";
                const isCreate = true;
                const formId = isCreate ? DUMMY_ID : objectId;
                const formData = { id: objectId, name: "Form Object" };
                setCookieFormData(objectType, formId, formData);

                const result = getStoredFormData({
                    objectType,
                    isCreate,
                    urlParams: { id: objectId },
                });
                expect(result).toEqual(formData);
            });
            test("update forms", () => {
                const objectType = "TestObject" as ModelType;
                const objectId = "12345";
                const isCreate = false;
                const formId = isCreate ? DUMMY_ID : objectId;
                const formData = { id: objectId, name: "Form Object" };
                setCookieFormData(objectType, formId, formData);

                const result = getStoredFormData({
                    objectType,
                    isCreate,
                    urlParams: { id: objectId },
                });
                expect(result).toEqual(formData);
            });
        });

        it("returns undefined when no stored form data is available", () => {
            // Store irrelevant form data to make sure it's not returned
            setCookieFormData("OtherObject" as ModelType, "12345", { id: "12345", name: "Irrelevant Object" });

            const result = getStoredFormData({
                objectType: "TestObject" as ModelType,
                isCreate: true,
                urlParams: { id: "12345" },
            });
            expect(result).toBeUndefined();

            const result2 = getStoredFormData({
                objectType: "TestObject" as ModelType,
                isCreate: false,
                urlParams: { id: "12345" },
            });
            expect(result2).toBeUndefined();
        });
    });

    describe("promptToUseStoredFormData", () => {
        let originalPubSubMethods;
        beforeEach(() => {
            originalPubSubMethods = { ...PubSub };
            Object.assign(PubSub, PubSubMock);
            PubSubMock.resetMock();
            jest.clearAllMocks();
        });

        afterEach(() => {
            Object.assign(PubSub, originalPubSubMethods);
        });

        it("publishes snack with correct parameters", () => {
            const mockSubscriber = jest.fn();
            PubSub.get().subscribe("snack", mockSubscriber);

            const onConfirm = jest.fn();
            promptToUseStoredFormData(onConfirm);
            expect(mockSubscriber).toHaveBeenCalledWith({
                autoHideDuration: "persist",
                messageKey: "FormDataFound",
                buttonKey: "Yes",
                buttonClicked: onConfirm,
                severity: "Warning",
            });
        });
    });
});
