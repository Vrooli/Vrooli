import { Grid } from "@mui/material";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { MarkdownStandardInputProps } from "../types";

/**
 * Input for entering (and viewing format of) Markdown data that 
 * must match a certain schema.
 */
export const MarkdownStandardInput = ({
    isEditing,
    zIndex,
}: MarkdownStandardInputProps) => {
    return (
        <Grid container spacing={2}>
            <Grid item xs={12}>
                <MarkdownInput
                    disabled={!isEditing}
                    name="defaultValue"
                    placeholder="Default value"
                    minRows={3}
                    zIndex={zIndex}
                />
            </Grid>
        </Grid>
    );
};
