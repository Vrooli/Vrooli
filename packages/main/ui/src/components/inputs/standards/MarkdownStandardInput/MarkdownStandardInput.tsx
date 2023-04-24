import { Grid } from "@mui/material";
import { MarkdownInput } from "../../MarkdownInput/MarkdownInput";
import { MarkdownStandardInputProps } from "../types";

/**
 * Input for entering (and viewing format of) Markdown data that 
 * must match a certain schema.
 */
export const MarkdownStandardInput = ({
    isEditing,
}: MarkdownStandardInputProps) => {
    return (
        <Grid container spacing={2}>
            <Grid item xs={12}>
                <MarkdownInput
                    disabled={!isEditing}
                    name="defaultValue"
                    placeholder="Default value"
                    minRows={3}
                />
            </Grid>
        </Grid>
    );
};