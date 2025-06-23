import { Button } from "../../buttons/Button.js";
import { type FormDividerProps } from "./types.js";

export function FormDivider({
    isEditing,
    onDelete,
}: FormDividerProps) {
    // If not selected, return a plain divider
    if (!isEditing) {
        return (
            <div 
                className="tw-w-full tw-h-0.5 tw-bg-text-secondary tw-opacity-20"
                data-testid="form-divider"
                data-editing="false"
                role="separator"
                aria-orientation="horizontal"
            />
        );
    }
    // Otherwise, return a divider with a delete button
    return (
        <div 
            className="tw-relative tw-flex tw-items-center tw-py-2"
            data-testid="form-divider"
            data-editing="true"
            role="separator"
            aria-orientation="horizontal"
        >
            <div className="tw-flex-grow tw-h-0.5 tw-bg-text-secondary tw-opacity-20" />
            <Button 
                variant="ghost" 
                size="sm" 
                onClick={onDelete}
                className="tw-mx-2"
                data-testid="form-divider-delete"
                aria-label="Delete divider"
            >
                Delete
            </Button>
            <div className="tw-flex-grow tw-h-0.5 tw-bg-text-secondary tw-opacity-20" />
        </div>
    );
}
