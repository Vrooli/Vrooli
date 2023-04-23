import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { BuildIcon, VisibleIcon } from "@local/icons";
import { Box, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useState } from "react";
import { InputTypeOptions } from "../../../../utils/consts";
import { GeneratedInputComponent } from "../../generated";
import { SelectorBase } from "../../SelectorBase/SelectorBase";
import { ToggleSwitch } from "../../ToggleSwitch/ToggleSwitch";
import { BaseStandardInput } from "../BaseStandardInput/BaseStandardInput";
export const StandardInput = ({ disabled = false, fieldName, zIndex, }) => {
    const { palette } = useTheme();
    const [field, , helpers] = useField(fieldName);
    const [isPreviewOn, setIsPreviewOn] = useState(false);
    const onPreviewChange = useCallback((event) => { setIsPreviewOn(event.target.checked); }, []);
    const [inputType, setInputType] = useState(InputTypeOptions[1]);
    const handleInputTypeSelect = useCallback((selected) => {
        setInputType(selected);
    }, []);
    const [schemaKey] = useState(`standard-create-schema-preview-${Math.random().toString(36).substring(2, 15)}`);
    console.log("standardinput field value", fieldName, field.value);
    return (_jsxs(Box, { sx: {
            background: palette.background.paper,
            borderRadius: 2,
            padding: 2,
        }, children: [!disabled && _jsx(ToggleSwitch, { checked: isPreviewOn, onChange: onPreviewChange, OffIcon: BuildIcon, OnIcon: VisibleIcon, label: isPreviewOn ? "Preview" : "Edit", tooltip: isPreviewOn ? "Switch to edit" : "Switch to preview", sx: { marginBottom: 2 } }), (isPreviewOn || disabled) ?
                (field.value && _jsx(GeneratedInputComponent, { disabled: true, fieldData: field.value, onUpload: () => { }, zIndex: zIndex })) :
                _jsxs(Box, { children: [_jsx(SelectorBase, { fullWidth: true, options: InputTypeOptions, value: inputType, onChange: handleInputTypeSelect, getOptionLabel: (option) => option.label, name: "inputType", inputAriaLabel: 'input-type-selector', label: "Type", sx: { marginBottom: 2 } }), _jsx(BaseStandardInput, { fieldName: fieldName, inputType: inputType.value, isEditing: true, storageKey: schemaKey })] })] }));
};
//# sourceMappingURL=StandardInput.js.map