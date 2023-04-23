import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Grid, TextField } from "@mui/material";
import { useTranslation } from "react-i18next";
import { IntegerInput } from "../../IntegerInput/IntegerInput";
export const TextFieldStandardInput = ({ isEditing, }) => {
    const { t } = useTranslation();
    return (_jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 12, children: _jsx(TextField, { fullWidth: true, disabled: !isEditing, name: "defaultValue", label: "Default Value", multiline: true }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(IntegerInput, { disabled: !isEditing, label: t("MaxRows"), name: "maxRows", tooltip: "The maximum number of rows to display" }) })] }));
};
//# sourceMappingURL=TextFieldStandardInput.js.map