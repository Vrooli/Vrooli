import { jsx as _jsx } from "react/jsx-runtime";
import { Grid } from "@mui/material";
import { MarkdownInput } from "../../MarkdownInput/MarkdownInput";
export const MarkdownStandardInput = ({ isEditing, }) => {
    return (_jsx(Grid, { container: true, spacing: 2, children: _jsx(Grid, { item: true, xs: 12, children: _jsx(MarkdownInput, { disabled: !isEditing, name: "defaultValue", placeholder: "Default value", minRows: 3 }) }) }));
};
//# sourceMappingURL=MarkdownStandardInput.js.map