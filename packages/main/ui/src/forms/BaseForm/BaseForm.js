import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { exists } from "@local/utils";
import { Form } from "formik";
import { forwardRef, useCallback, useEffect, useImperativeHandle } from "react";
import { usePromptBeforeUnload } from "../../utils/hooks/usePromptBeforeUnload";
export const BaseForm = forwardRef(({ children, dirty, isLoading = false, promptBeforeUnload = true, style, }, ref) => {
    useEffect(() => {
        if (promptBeforeUnload && !exists(dirty)) {
            console.warn("BaseForm: promptBeforeUnload is true but dirty is not defined. This will cause the prompt to never appear.");
        }
    }, [promptBeforeUnload, dirty]);
    usePromptBeforeUnload({ shouldPrompt: promptBeforeUnload && dirty });
    const handleClose = useCallback((onClose, closeAnyway) => {
        if (dirty && closeAnyway !== true) {
            if (window.confirm("You have unsaved changes. Are you sure you want to close?")) {
                onClose();
            }
        }
        else {
            onClose();
        }
    }, [dirty]);
    useImperativeHandle(ref, () => ({ handleClose }));
    return (_jsxs(Form, { style: {
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            ...(style ?? {}),
        }, children: [isLoading && _jsx("div", { style: {
                    position: "absolute",
                    top: 0,
                    left: 0,
                    width: "100%",
                    height: "100%",
                    backgroundColor: "rgba(0, 0, 0, 0.2)",
                    zIndex: 1,
                } }), children] }));
});
//# sourceMappingURL=BaseForm.js.map