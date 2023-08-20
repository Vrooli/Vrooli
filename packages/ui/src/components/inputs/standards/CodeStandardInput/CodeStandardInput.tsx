import { Grid } from "@mui/material";
import { CodeInput } from "components/inputs/CodeInput/CodeInput";
import { CodeStandardInputProps } from "../types";

/**
 * Input for entering (and viewing format of) code (e.g. JSON, TypeScript, Solidity)
 */
export const CodeStandardInput = ({
    isEditing,
    fieldName,
    ...props
}: CodeStandardInputProps) => {
    console.log("rendering code input", props);
    return (
        <Grid container spacing={2}>
            <Grid item xs={12}>
                <CodeInput
                    disabled={!isEditing}
                    name={fieldName}
                    {...props}
                />
            </Grid>
        </Grid>
    );
};
