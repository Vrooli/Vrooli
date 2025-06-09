import { type FormikProps } from "formik";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi, afterEach } from "vitest";
import { useAutoSave } from "./useAutoSave.js";
import { useDebounce } from "./useDebounce.js";

// Mock useDebounce hook
vi.mock("./useDebounce.js");

describe("useAutoSave", () => {
    let mockFormikRef: { current: FormikProps<any> | null };
    let mockHandleSave: ReturnType<typeof vi.fn>;
    let mockDebouncedSave: ReturnType<typeof vi.fn>;
    let mockCancelDebounce: ReturnType<typeof vi.fn>;

    beforeEach(() => {
        vi.clearAllMocks();
        vi.useFakeTimers();

        mockHandleSave = vi.fn();
        mockDebouncedSave = vi.fn();
        mockCancelDebounce = vi.fn();

        // Mock useDebounce to return our mock functions
        vi.mocked(useDebounce).mockReturnValue([mockDebouncedSave, mockCancelDebounce]);

        // Create a mock formik ref
        mockFormikRef = {
            current: {
                values: { name: "test", email: "test@example.com" },
                dirty: false,
                isSubmitting: false,
                setFieldValue: vi.fn(),
                resetForm: vi.fn(),
                submitForm: vi.fn(),
            } as unknown as FormikProps<any>,
        };
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.clearAllTimers();
    });

    describe("initialization", () => {
        it("should set up debounced save with default debounce time", () => {
            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            expect(useDebounce).toHaveBeenCalledWith(expect.any(Function), 2000);
        });

        it("should set up debounced save with custom debounce time", () => {
            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                    debounceMs: 5000,
                })
            );

            expect(useDebounce).toHaveBeenCalledWith(expect.any(Function), 5000);
        });

        it("should start interval for checking form changes", () => {
            const setIntervalSpy = vi.spyOn(global, "setInterval");
            
            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            expect(setIntervalSpy).toHaveBeenCalledWith(expect.any(Function), 250);
        });
    });

    describe("auto-save trigger conditions", () => {
        it("should trigger debounced save when form becomes dirty and values change", () => {
            mockFormikRef.current!.dirty = true;

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            // Simulate value change
            act(() => {
                mockFormikRef.current!.values = { name: "changed", email: "test@example.com" };
                vi.advanceTimersByTime(250); // Trigger interval check
            });

            expect(mockDebouncedSave).toHaveBeenCalled();
        });

        it("should not trigger save when form is not dirty", () => {
            mockFormikRef.current!.dirty = false;

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            act(() => {
                mockFormikRef.current!.values = { name: "changed", email: "test@example.com" };
                vi.advanceTimersByTime(250);
            });

            expect(mockDebouncedSave).not.toHaveBeenCalled();
        });

        it("should not trigger save when form is submitting", () => {
            mockFormikRef.current!.dirty = true;
            mockFormikRef.current!.isSubmitting = true;

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            act(() => {
                mockFormikRef.current!.values = { name: "changed", email: "test@example.com" };
                vi.advanceTimersByTime(250);
            });

            expect(mockDebouncedSave).not.toHaveBeenCalled();
        });

        it("should not trigger save when disabled", () => {
            mockFormikRef.current!.dirty = true;

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                    disabled: true,
                })
            );

            act(() => {
                mockFormikRef.current!.values = { name: "changed", email: "test@example.com" };
                vi.advanceTimersByTime(250);
            });

            expect(mockDebouncedSave).not.toHaveBeenCalled();
        });

        it("should handle null formik ref gracefully", () => {
            const nullFormikRef = { current: null };

            renderHook(() =>
                useAutoSave({
                    formikRef: nullFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            act(() => {
                vi.advanceTimersByTime(250);
            });

            expect(mockDebouncedSave).not.toHaveBeenCalled();
        });
    });

    describe("maybeSave function", () => {
        it("should call handleSave when conditions are met", () => {
            mockFormikRef.current!.dirty = true;
            mockFormikRef.current!.isSubmitting = false;

            // Mock the debounced function to call maybeSave directly
            const maybeSave = vi.fn();
            useDebounce.mockReturnValue([maybeSave, mockCancelDebounce]);

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            // Get the maybeSave function that was passed to useDebounce
            const maybeSaveFunction = useDebounce.mock.calls[0][0];
            
            act(() => {
                maybeSaveFunction();
            });

            expect(mockHandleSave).toHaveBeenCalled();
        });

        it("should not call handleSave when disabled", () => {
            mockFormikRef.current!.dirty = true;

            const maybeSave = vi.fn();
            useDebounce.mockReturnValue([maybeSave, mockCancelDebounce]);

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                    disabled: true,
                })
            );

            const maybeSaveFunction = useDebounce.mock.calls[0][0];
            
            act(() => {
                maybeSaveFunction();
            });

            expect(mockHandleSave).not.toHaveBeenCalled();
        });

        it("should not call handleSave when form is not dirty", () => {
            mockFormikRef.current!.dirty = false;

            const maybeSave = vi.fn();
            useDebounce.mockReturnValue([maybeSave, mockCancelDebounce]);

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            const maybeSaveFunction = useDebounce.mock.calls[0][0];
            
            act(() => {
                maybeSaveFunction();
            });

            expect(mockHandleSave).not.toHaveBeenCalled();
        });

        it("should not call handleSave when form is submitting", () => {
            mockFormikRef.current!.dirty = true;
            mockFormikRef.current!.isSubmitting = true;

            const maybeSave = vi.fn();
            useDebounce.mockReturnValue([maybeSave, mockCancelDebounce]);

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            const maybeSaveFunction = useDebounce.mock.calls[0][0];
            
            act(() => {
                maybeSaveFunction();
            });

            expect(mockHandleSave).not.toHaveBeenCalled();
        });
    });

    describe("unsaved changes detection", () => {
        it("should detect unsaved changes when no previous save exists", () => {
            mockFormikRef.current!.dirty = true;

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            act(() => {
                mockFormikRef.current!.values = { name: "changed", email: "test@example.com" };
                vi.advanceTimersByTime(250);
            });

            expect(mockDebouncedSave).toHaveBeenCalled();
        });

        it("should not trigger save when values haven't changed from last save", () => {
            mockFormikRef.current!.dirty = true;

            // First, trigger a save to establish "last saved values"
            const maybeSave = vi.fn();
            useDebounce.mockReturnValue([maybeSave, mockCancelDebounce]);

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            // Simulate saving current values
            const maybeSaveFunction = useDebounce.mock.calls[0][0];
            act(() => {
                maybeSaveFunction();
            });

            // Clear the mock to check for new calls
            mockDebouncedSave.mockClear();

            // Now change values back to the same state
            act(() => {
                mockFormikRef.current!.values = { name: "test", email: "test@example.com" };
                mockFormikRef.current!.dirty = false; // Formik would set this
                vi.advanceTimersByTime(250);
            });

            expect(mockDebouncedSave).not.toHaveBeenCalled();
        });
    });

    describe("rapid changes handling", () => {
        it("should handle multiple rapid value changes correctly", () => {
            mockFormikRef.current!.dirty = true;

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            // Simulate rapid changes
            act(() => {
                mockFormikRef.current!.values = { name: "change1", email: "test@example.com" };
                vi.advanceTimersByTime(100);
                
                mockFormikRef.current!.values = { name: "change2", email: "test@example.com" };
                vi.advanceTimersByTime(100);
                
                mockFormikRef.current!.values = { name: "change3", email: "test@example.com" };
                vi.advanceTimersByTime(100);
            });

            // Should call debounced save for each detected change
            expect(mockDebouncedSave).toHaveBeenCalledTimes(3);
        });
    });

    describe("beforeunload event handling", () => {
        it("should add beforeunload event listener", () => {
            const addEventListenerSpy = vi.spyOn(window, "addEventListener");

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            expect(addEventListenerSpy).toHaveBeenCalledWith("beforeunload", expect.any(Function));
        });

        it("should save on beforeunload when there are unsaved changes", () => {
            mockFormikRef.current!.dirty = true;

            const { unmount } = renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            // Simulate beforeunload event
            const beforeUnloadEvent = new Event("beforeunload") as any;
            beforeUnloadEvent.preventDefault = vi.fn();

            act(() => {
                window.dispatchEvent(beforeUnloadEvent);
            });

            expect(mockCancelDebounce).toHaveBeenCalled();
            expect(mockHandleSave).toHaveBeenCalled();

            unmount();
        });

        it("should not save on beforeunload when disabled", () => {
            mockFormikRef.current!.dirty = true;

            const { unmount } = renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                    disabled: true,
                })
            );

            const beforeUnloadEvent = new Event("beforeunload") as any;
            beforeUnloadEvent.preventDefault = vi.fn();

            act(() => {
                window.dispatchEvent(beforeUnloadEvent);
            });

            expect(mockHandleSave).not.toHaveBeenCalled();

            unmount();
        });

        it("should remove beforeunload event listener on unmount", () => {
            const removeEventListenerSpy = vi.spyOn(window, "removeEventListener");

            const { unmount } = renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            unmount();

            expect(removeEventListenerSpy).toHaveBeenCalledWith("beforeunload", expect.any(Function));
        });
    });

    describe("component unmount handling", () => {
        it("should save on unmount when there are unsaved changes", () => {
            mockFormikRef.current!.dirty = true;

            // Trigger a value change to make it have unsaved changes
            const { unmount } = renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            act(() => {
                mockFormikRef.current!.values = { name: "changed", email: "test@example.com" };
            });

            unmount();

            expect(mockCancelDebounce).toHaveBeenCalled();
            expect(mockHandleSave).toHaveBeenCalled();
        });

        it("should not save on unmount when form is not dirty", () => {
            mockFormikRef.current!.dirty = false;

            const { unmount } = renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            unmount();

            expect(mockHandleSave).not.toHaveBeenCalled();
        });

        it("should not save on unmount when disabled", () => {
            mockFormikRef.current!.dirty = true;

            const { unmount } = renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                    disabled: true,
                })
            );

            unmount();

            expect(mockHandleSave).not.toHaveBeenCalled();
        });

        it("should not save on unmount when form is submitting", () => {
            mockFormikRef.current!.dirty = true;
            mockFormikRef.current!.isSubmitting = true;

            const { unmount } = renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            unmount();

            expect(mockHandleSave).not.toHaveBeenCalled();
        });
    });

    describe("cleanup", () => {
        it("should clear interval on unmount", () => {
            const clearIntervalSpy = vi.spyOn(global, "clearInterval");

            const { unmount } = renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            unmount();

            expect(clearIntervalSpy).toHaveBeenCalled();
        });
    });

    describe("edge cases", () => {
        it("should handle formik ref becoming null after initialization", () => {
            const { rerender } = renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            // Set formik ref to null
            mockFormikRef.current = null;

            act(() => {
                vi.advanceTimersByTime(250);
            });

            expect(mockDebouncedSave).not.toHaveBeenCalled();
        });

        it("should handle handleSave function changing", () => {
            const newHandleSave = vi.fn();

            const { rerender } = renderHook(
                ({ handleSave }) =>
                    useAutoSave({
                        formikRef: mockFormikRef,
                        handleSave,
                    }),
                {
                    initialProps: { handleSave: mockHandleSave },
                }
            );

            // Change the handleSave function
            rerender({ handleSave: newHandleSave });

            mockFormikRef.current!.dirty = true;

            act(() => {
                mockFormikRef.current!.values = { name: "changed", email: "test@example.com" };
                vi.advanceTimersByTime(250);
            });

            expect(mockDebouncedSave).toHaveBeenCalled();
        });

        it("should handle complex object changes correctly", () => {
            mockFormikRef.current!.dirty = true;
            mockFormikRef.current!.values = {
                user: { name: "test", email: "test@example.com" },
                settings: { theme: "dark", notifications: true },
                nested: { deep: { value: "original" } },
            };

            renderHook(() =>
                useAutoSave({
                    formikRef: mockFormikRef,
                    handleSave: mockHandleSave,
                })
            );

            act(() => {
                mockFormikRef.current!.values = {
                    user: { name: "test", email: "test@example.com" },
                    settings: { theme: "light", notifications: true }, // Changed theme
                    nested: { deep: { value: "original" } },
                };
                vi.advanceTimersByTime(250);
            });

            expect(mockDebouncedSave).toHaveBeenCalled();
        });
    });
});