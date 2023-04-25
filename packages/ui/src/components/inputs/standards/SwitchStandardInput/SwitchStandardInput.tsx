
import { FormControlLabel, FormGroup, Switch } from "@mui/material";
import { useField } from "formik";
import { SwitchStandardInputProps } from "../types";

/**
 * Input for entering (and viewing format of) Switch data that 
 * must match a certain schema.
 */
export const SwitchStandardInput = ({
    isEditing,
}: SwitchStandardInputProps) => {
    const [defaultValueField, , defaulValueHelpers] = useField<boolean>("defaultValue");

    return (
        <FormGroup>
            <FormControlLabel control={(
                <Switch
                    disabled={!isEditing}
                    size={"medium"}
                    color="secondary"
                    checked={defaultValueField.value}
                    onChange={(event) => {
                        defaulValueHelpers.setValue(event.target.checked);
                    }}
                />
            )} label="Default checked" />
        </FormGroup>
    );
};
