import type React from "react";
import { TOOLBAR_CLASS_NAME } from "../AdvancedInputToolbar.js";

// Simple interface for elements that can be focused
export interface FocusableElement {
    focus: (options?: FocusOptions) => void;
}

/** Prevents input from losing focus when the toolbar is pressed */
export function preventInputLossOnToolbarClick(event: React.MouseEvent) {
    // Check for the toolbar ID at each parent element
    let parent = event.target as HTMLElement | null | undefined;
    let numParentsTraversed = 0;
    const maxParentsToTraverse = 10;
    do {
        // If the toolbar is clicked, prevent default (stops focus loss) and stop propagation (prevents outer click handler)
        if (parent?.classList.contains(TOOLBAR_CLASS_NAME)) {
            event.preventDefault();
            // We don't stop propagation here anymore, let the Outer click handler decide based on selector
            // event.stopPropagation();
            return;
        }
        parent = parent?.parentElement;
        numParentsTraversed++;
    } while (parent && numParentsTraversed < maxParentsToTraverse);
}

/**
 * Handles the drop event for the drag and drop feature.
 * @param event The drop event
 * @returns An array of file entries extracted from the drop event
 */
export async function getFilesFromEvent(event: React.DragEvent<HTMLDivElement> | undefined): Promise<File[]> {
    if (!event?.dataTransfer?.items) return [];

    const files: File[] = [];
    const items = Array.from(event.dataTransfer.items);

    for (const item of items) {
        if (item.kind === "file") {
            const file = item.getAsFile();
            if (file) {
                files.push(file);
            }
        }
    }

    return files;
}
