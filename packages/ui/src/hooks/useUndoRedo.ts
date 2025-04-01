import { useCallback, useRef, useState } from "react";
import { useDebounce } from "./useDebounce.js";

type UpdaterFn<T> = (prev: T) => T;

interface UseUndoRedoParams<T> {
    disabled?: boolean;
    initialValue: T;
    onChange: (updated: T) => unknown;
    /**
     * Optional array of delimiters that will force-add to the stack immediately when typed.
     * For example, [" ", "\n"] will add to the stack immediately when a space or newline is typed.
     * This helps prevent losing a lot of text when undoing after typing consistently.
     * 
     * If you want to disable debouncing entirely (i.e. add to stack immediately for every change),
     * pass an empty array [].
     */
    delimiters?: string[];
}

/**
 * Total characters stored in the change stack. 
 * 1 million characters will be about 1-1.5 MB
 */
const MAX_STACK_SIZE = 1000000;
const DEBOUNCE_TIME_MS = 200;

/**
 * Adds debounced undo/redo functionality to a value.
 */
export function useUndoRedo<T>({
    disabled,
    initialValue,
    onChange,
    delimiters,
}: UseUndoRedoParams<T>) {
    const [internalValue, setInternalValue] = useState(initialValue);
    const [onChangeDebounced, cancelDebounce] = useDebounce(onChange, DEBOUNCE_TIME_MS);

    const stack = useRef([initialValue]);
    const stackIndex = useRef(0);
    const stackSize = useRef(JSON.stringify(initialValue)?.length ?? 0);

    const [canUndo, setCanUndo] = useState(false);
    const [canRedo, setCanRedo] = useState(false);
    const updateUndoRedoStates = useCallback(() => {
        setCanUndo(stackIndex.current > 0);
        setCanRedo(stackIndex.current < stack.current.length - 1);
    }, []);

    const addToStack = useCallback((newValue: T) => {
        const newStack = [...stack.current];
        // Remove any future states when adding new state
        newStack.splice(stackIndex.current + 1, newStack.length - stackIndex.current - 1);
        newStack.push(newValue);

        // If the stack is too big, remove the oldest items
        while (stackSize.current > MAX_STACK_SIZE && newStack.length > 1) {
            stackSize.current -= JSON.stringify(newStack[0]).length;
            newStack.shift();
            // Adjust stackIndex when removing items from the beginning
            stackIndex.current = Math.max(0, stackIndex.current - 1);
        }

        // Add the new value's size
        stackSize.current += JSON.stringify(newValue).length;

        stack.current = newStack;
        stackIndex.current = newStack.length - 1;
        updateUndoRedoStates();
    }, [updateUndoRedoStates]);

    const [addToStackDebounced, cancelAddToStack] = useDebounce(addToStack, DEBOUNCE_TIME_MS);
    const changeInternalValue = useCallback((newValueOrUpdater: T | UpdaterFn<T>) => {
        setInternalValue((prev) => {
            const newValue = typeof newValueOrUpdater === "function"
                ? (newValueOrUpdater as UpdaterFn<T>)(prev)
                : newValueOrUpdater;

            onChangeDebounced(newValue);

            // If delimiters is an empty array, disable debouncing entirely
            if (Array.isArray(delimiters) && delimiters.length === 0) {
                cancelAddToStack();
                addToStack(newValue);
                return newValue;
            }

            // By default, start debounced addition
            addToStackDebounced(newValue);

            // If delimiters are provided and the value is a string, check if we should add immediately
            if (delimiters?.length && typeof newValue === "string" && typeof prev === "string") {
                // Only check the last character if the new value is longer than the previous value
                if (newValue.length > prev.length) {
                    const lastChar = newValue[newValue.length - 1];
                    if (delimiters.includes(lastChar)) {
                        // Cancel the debounced action and add immediately
                        cancelAddToStack();
                        addToStack(newValue);
                    }
                }
            }

            return newValue;
        });
    }, [addToStack, addToStackDebounced, cancelAddToStack, delimiters, onChangeDebounced]);

    /** 
     * Sets internal value without adding to the stack or 
     * debouncing the onChange callback. Also cancels any
     * pending debounced calls.
     */
    const resetInternalValue = useCallback((newValue: T) => {
        setInternalValue(newValue);
        cancelDebounce();
    }, [cancelDebounce]);

    /**
    * Clears the undo/redo stack and resets it to the current internal value.
    * Also cancels any pending debounced additions.
    */
    const resetStack = useCallback(() => {
        cancelDebounce();
        cancelAddToStack();

        const currentValue = internalValue;
        stack.current = [currentValue];
        stackIndex.current = 0;
        stackSize.current = JSON.stringify(currentValue).length;

        updateUndoRedoStates();
    }, [internalValue, cancelDebounce, cancelAddToStack, updateUndoRedoStates]);

    function undo() {
        if (disabled === true || stackIndex.current <= 0) return;
        stackIndex.current -= 1;
        setInternalValue(stack.current[stackIndex.current]);
        onChangeDebounced(stack.current[stackIndex.current]);
        updateUndoRedoStates();
    }

    function redo() {
        if (disabled === true || stackIndex.current >= stack.current.length - 1) return;
        stackIndex.current += 1;
        setInternalValue(stack.current[stackIndex.current]);
        onChangeDebounced(stack.current[stackIndex.current]);
        updateUndoRedoStates();
    }

    return {
        internalValue,
        changeInternalValue,
        resetInternalValue,
        resetStack,
        undo,
        redo,
        canUndo,
        canRedo,
        stackSize,
    };
}
