import { useDebounce } from "hooks/useDebounce";
import { useCallback, useRef, useState } from "react";

/**
 * Total characters stored in the change stack. 
 * 1 million characters will be about 1-1.5 MB
 */
const MAX_STACK_SIZE = 1000000;
const DEBOUNCE_TIME = 200;

/**
 * Adds debounced undo/redo functionality to a value.
 */
export const useUndoRedo = <T>({
    disabled,
    initialValue,
    onChange,
    forceAddToStack,
}: {
    disabled?: boolean;
    initialValue: T;
    onChange: (updated: T) => unknown;
    /** Determines when the stack should be updated */
    forceAddToStack?: (updated: T, resetStackDebounce: () => unknown) => boolean;
}) => {
    const [internalValue, setInternalValue] = useState(initialValue);
    const [onChangeDebounced, cancelDebounce] = useDebounce(onChange, DEBOUNCE_TIME);

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
        newStack.splice(stackIndex.current + 1, newStack.length - stackIndex.current - 1);
        newStack.push(newValue);
        stackSize.current += JSON.stringify(newValue).length;

        // If the stack is too big, remove the oldest items
        while (stackSize.current > MAX_STACK_SIZE) {
            stackSize.current -= JSON.stringify(newStack[0]).length;
            newStack.shift();
        }

        stack.current = newStack;
        stackIndex.current = newStack.length - 1;
        updateUndoRedoStates();
    }, [updateUndoRedoStates]);

    const [addToStackDebounced, cancelAddToStack] = useDebounce(addToStack, DEBOUNCE_TIME);
    const changeInternalValue = useCallback((newValue: T) => {
        setInternalValue(newValue);
        onChangeDebounced(newValue);
        // By default, start debounced addition
        addToStackDebounced(newValue);
        // If the forceAddToStack callback is provided, call it
        if (forceAddToStack) {
            // Determines if we should skip debouncing and add to the stack immediately. 
            // Provides resetAddToStack callback to cancel the debounced action, for things 
            // like updating stack based on inactivity.
            const shouldImmediatelyAdd = forceAddToStack(newValue, () => {
                cancelAddToStack();
                addToStackDebounced(newValue);
            });
            if (shouldImmediatelyAdd) {
                // Cancel the debounced action and add to the stack immediately
                cancelAddToStack();
                addToStack(newValue);
            }
        }
    }, [addToStack, addToStackDebounced, cancelAddToStack, onChangeDebounced, forceAddToStack]);

    /** 
     * Sets internal value without adding to the stack or 
     * debouncing the onChange callback. Also cancels any
     * pending debounced calls.
     **/
    const resetInternalValue = useCallback((newValue: T) => {
        setInternalValue(newValue);
        cancelDebounce();
    }, [cancelDebounce]);

    const undo = () => {
        if (disabled === true || stackIndex.current <= 0) return;
        stackIndex.current -= 1;
        setInternalValue(stack.current[stackIndex.current]);
        onChangeDebounced(stack.current[stackIndex.current]);
        updateUndoRedoStates();
    };

    const redo = () => {
        if (disabled === true || stackIndex.current >= stack.current.length - 1) return;
        stackIndex.current += 1;
        setInternalValue(stack.current[stackIndex.current]);
        onChangeDebounced(stack.current[stackIndex.current]);
        updateUndoRedoStates();
    };

    return {
        internalValue,
        changeInternalValue,
        resetInternalValue,
        undo,
        redo,
        canUndo,
        canRedo,
    };
};
