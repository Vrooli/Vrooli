import type { FormViewProps } from "@vrooli/shared";
import { FormBuildView } from "./FormBuildView.js";
import { FormRunView } from "./FormRunView.js";

// Extended props to support testing attributes
export interface FormViewTestProps extends FormViewProps {
    "data-testid"?: string;
}

//TODO: Allow titles to collapse sections below them. Need to update the way inputs are rendered so that they're nested within the header using a collapsibetext component
export function FormView({
    isEditing,
    "data-testid": dataTestId,
    ...props
}: FormViewTestProps) {
    if (isEditing) {
        return <div data-testid={dataTestId || "form-view"}><FormBuildView {...props} /></div>;
    }
    return <div data-testid={dataTestId || "form-view"}><FormRunView {...props} /></div>;
}

// Re-export utility functions and components that might be used elsewhere
export { normalizeFormContainers, calculateGridItemSize } from "./FormView.utils.js";
export { GeneratedGridItem } from "./GeneratedGridItem.js";

// Re-export sub-components for backward compatibility
export { FormBuildView } from "./FormBuildView.js";
export { FormRunView } from "./FormRunView.js";
