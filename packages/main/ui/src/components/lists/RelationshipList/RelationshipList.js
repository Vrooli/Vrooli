import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Stack, useTheme } from "@mui/material";
import { noSelect } from "../../../styles";
import { getCurrentUser } from "../../../utils/authentication/session";
import { IsCompleteButton, IsPrivateButton, OwnerButton, ParentButton, ProjectButton } from "../../buttons/relationships";
export const userFromSession = (session) => ({
    __typename: "User",
    id: getCurrentUser(session).id,
    handle: null,
    name: "Self",
});
export function RelationshipList(props) {
    const { palette } = useTheme();
    return (_jsxs(Stack, { spacing: { xs: 1, sm: 1.5, md: 2 }, direction: "row", alignItems: "center", justifyContent: "center", p: 1, sx: {
            borderRadius: 2,
            background: palette.mode === "dark" ? palette.background.paper : palette.background.default,
            overflowX: "auto",
            zIndex: props.zIndex,
            ...noSelect,
            ...(props.sx ?? {}),
        }, children: [_jsx(OwnerButton, { ...props }), _jsx(ProjectButton, { ...props }), _jsx(ParentButton, { ...props }), _jsx(IsPrivateButton, { ...props }), _jsx(IsCompleteButton, { ...props })] }));
}
//# sourceMappingURL=RelationshipList.js.map