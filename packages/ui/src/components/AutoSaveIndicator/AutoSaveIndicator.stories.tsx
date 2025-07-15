import { Box, Button, TextField } from "@mui/material";
import { Formik, type FormikProps } from "formik";
import { useRef, useState } from "react";
import { pageDecorator } from "../../__test/helpers/storybookDecorators.js";
import { useAutoSave } from "../../hooks/useAutoSave.js";
import { AutoSaveIndicator } from "./AutoSaveIndicator.js";

export default {
    title: "Components/AutoSaveIndicator",
    component: AutoSaveIndicator,
    decorators: [pageDecorator],
};

interface FormValues {
    firstName: string;
    lastName: string;
    email: string;
}

export function InteractiveAutoSave() {
    const formikRef = useRef<FormikProps<FormValues>>(null);
    const [saveCount, setSaveCount] = useState(0);
    const [lastSavedData, setLastSavedData] = useState<FormValues | null>(null);

    // Simulate an async save operation
    const handleSave = async () => {
        if (!formikRef.current) return;
        
        // Simulate API call delay
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        const values = formikRef.current.values;
        setLastSavedData(values);
        setSaveCount(prev => prev + 1);
        
        // Mark form as clean after successful save
        formikRef.current.setSubmitting(false);
        formikRef.current.resetForm({ values });
    };

    return (
        <Box 
            sx={{ 
                display: "flex", 
                flexDirection: "column", 
                alignItems: "center", 
                gap: 3,
                p: 3,
                mt: 4,
            }}
        >
            <Formik<FormValues>
                innerRef={formikRef}
                initialValues={{
                    firstName: "",
                    lastName: "",
                    email: "",
                }}
                onSubmit={handleSave}
            >
                {(formik) => {
                    // Use the auto-save hook
                    useAutoSave({
                        formikRef,
                        handleSave: () => formik.submitForm(),
                        debounceMs: 2000, // Save 2 seconds after user stops typing
                    });

                    return (
                        <Box 
                            component="form" 
                            onSubmit={formik.handleSubmit}
                            sx={{
                                display: "flex",
                                flexDirection: "column",
                                gap: 2,
                                width: "100%",
                                maxWidth: 400,
                                p: 3,
                                border: "1px solid",
                                borderColor: "divider",
                                borderRadius: 2,
                                bgcolor: "background.paper",
                            }}
                        >
                            <Box sx={{ alignSelf: "flex-end" }}>
                                <AutoSaveIndicator formikRef={formikRef} />
                            </Box>

                            <TextField
                                name="firstName"
                                label="First Name"
                                value={formik.values.firstName}
                                onChange={formik.handleChange}
                                onBlur={formik.handleBlur}
                                fullWidth
                            />

                            <TextField
                                name="lastName"
                                label="Last Name"
                                value={formik.values.lastName}
                                onChange={formik.handleChange}
                                onBlur={formik.handleBlur}
                                fullWidth
                            />

                            <TextField
                                name="email"
                                label="Email"
                                type="email"
                                value={formik.values.email}
                                onChange={formik.handleChange}
                                onBlur={formik.handleBlur}
                                fullWidth
                            />

                            <Button
                                type="submit"
                                variant="contained"
                                disabled={formik.isSubmitting}
                                fullWidth
                            >
                                Save Manually
                            </Button>
                        </Box>
                    );
                }}
            </Formik>

            <Box 
                sx={{ 
                    textAlign: "center",
                    p: 2,
                    bgcolor: "background.paper",
                    borderRadius: 1,
                    border: "1px solid",
                    borderColor: "divider",
                    width: "100%",
                    maxWidth: 400,
                }}
            >
                <Box sx={{ mb: 1 }}>
                    <strong>Save Count:</strong> {saveCount}
                </Box>
                {lastSavedData && (
                    <Box sx={{ textAlign: "left", fontSize: "0.875rem" }}>
                        <strong>Last Saved Data:</strong>
                        <pre style={{ margin: "8px 0", overflow: "auto" }}>
                            {JSON.stringify(lastSavedData, null, 2)}
                        </pre>
                    </Box>
                )}
            </Box>

            <Box sx={{ maxWidth: 400, textAlign: "center", color: "text.secondary" }}>
                <p>
                    Try typing in the form fields. The auto-save will trigger 2 seconds after you stop typing.
                    The indicator will show:
                </p>
                <ul style={{ textAlign: "left" }}>
                    <li><strong>Not saved</strong> (orange) - when you have unsaved changes</li>
                    <li><strong>Saving...</strong> (blue) - while saving is in progress</li>
                    <li><strong>Saved</strong> (green) - after successful save (disappears after 3 seconds)</li>
                </ul>
            </Box>
        </Box>
    );
}

InteractiveAutoSave.parameters = {
    docs: {
        description: {
            story: "Interactive example showing the AutoSaveIndicator with a working form. " +
                   "The form automatically saves 2 seconds after you stop typing. " +
                   "You can also save manually using the button. " +
                   "The save count and last saved data are displayed below the form.",
        },
    },
};