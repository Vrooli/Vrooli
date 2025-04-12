import { useCallback, useState } from "react";
import { FindObjectDialog } from "./FindObjectDialog.js";

export default {
    title: "Components/Dialogs/FindObjectDialog",
    component: FindObjectDialog,
};

export function Default() {
    const [isOpen, setIsOpen] = useState(true);
    const handleCancel = useCallback(() => setIsOpen(false), []);
    const handleComplete = useCallback(() => setIsOpen(false), []);

    return (
        <FindObjectDialog
            find="Full"
            handleCancel={handleCancel}
            handleComplete={handleComplete}
            isOpen={isOpen}
        />
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the default find object dialog.",
        },
    },
};

export function SingleObjectType() {
    const [isOpen, setIsOpen] = useState(true);
    const handleCancel = useCallback(() => setIsOpen(false), []);
    const handleComplete = useCallback(() => setIsOpen(false), []);
    const limitTo = ["Prompt"] as const;

    return (
        <FindObjectDialog
            find="Full"
            handleCancel={handleCancel}
            handleComplete={handleComplete}
            isOpen={isOpen}
            limitTo={limitTo}
        />
    );
}
SingleObjectType.parameters = {
    docs: {
        description: {
            story: "Displays the default find object dialog when limited to a single object type.",
        },
    },
};
