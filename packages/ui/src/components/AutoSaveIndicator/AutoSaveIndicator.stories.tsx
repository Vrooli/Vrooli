import { FormikProps } from "formik";
import { useRef } from "react";
import { AutoSaveIndicator } from "./AutoSaveIndicator.js";

export default {
    title: "Components/AutoSaveIndicator",
    component: AutoSaveIndicator,
};

const outerStyle = {
    width: "300px",
    padding: "20px",
    border: "1px solid #ccc",
} as const;
function Outer({ children }: { children: React.ReactNode }) {
    return (
        <div style={outerStyle}>
            {children}
        </div>
    );
}

export function Saving() {
    const formikRef = useRef({
        dirty: false,
        isSubmitting: true,
    } as FormikProps<object>);
    return (
        <Outer>
            <AutoSaveIndicator formikRef={formikRef} />
        </Outer>
    );
}
Saving.parameters = {
    docs: {
        description: {
            story: "Displays the \"Saving...\" status when the form is submitting.",
        },
    },
};

export function Saved() {
    const formikRef = useRef({
        dirty: false,
        isSubmitting: false,
    } as FormikProps<object>);
    return (
        <Outer>
            <AutoSaveIndicator formikRef={formikRef} savedIndicatorTimeoutMs={10_000} />
        </Outer>
    );
}
Saved.parameters = {
    docs: {
        description: {
            story: "Displays the \"Saved\" status briefly when the form is not dirty and not submitting. The indicator hides after 3 seconds, as per the component’s design.",
        },
    },
};

export function Unsaved() {
    const formikRef = useRef({
        dirty: true,
        isSubmitting: false,
    } as FormikProps<object>);
    return (
        <Outer>
            <AutoSaveIndicator formikRef={formikRef} />
        </Outer>
    );
}
Unsaved.parameters = {
    docs: {
        description: {
            story: "Displays the \"Not saved\" status when the form is dirty but not submitting.",
        },
    },
};
