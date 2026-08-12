/**
 * @vrooliComponentSource react-component-library:ColorPicker
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 623f00f8-2a74-40ec-83bc-67c4575b6cb6
 * @vrooliComponentAppliedAt 2026-07-22T16:50:28Z
 * @vrooliComponentSourceSha256 e7ef631d59b12836357930dc5d0e21247235456c5cc5fc4d0673e5ef2fd5ff62
 * @vrooliComponentDriftHash e7ef631d59b12836357930dc5d0e21247235456c5cc5fc4d0673e5ef2fd5ff62
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useCallback, useEffect, useRef } from "react";
/** Records a native-color value only after blur or component unmount. */
export function useDeferredColorCommit(onCommit) {
    const pending = useRef(null);
    const flush = useCallback(() => {
        const color = pending.current;
        pending.current = null;
        if (color)
            onCommit?.(color);
    }, [onCommit]);
    useEffect(() => flush, [flush]);
    return { park: (color) => { pending.current = color; }, flush };
}
