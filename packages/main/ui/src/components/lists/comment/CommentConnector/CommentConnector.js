import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { OpenThreadIcon, OrganizationIcon, UserIcon } from "@local/icons";
import { Box, IconButton, Stack, useTheme } from "@mui/material";
import { useMemo } from "react";
import { placeholderColor } from "../../../../utils/display/listTools";
export const CommentConnector = ({ isOpen, parentType, onToggle, }) => {
    const { palette } = useTheme();
    const profileColors = useMemo(() => placeholderColor(), []);
    const ProfileIcon = useMemo(() => {
        switch (parentType) {
            case "Organization":
                return OrganizationIcon;
            default:
                return UserIcon;
        }
    }, [parentType]);
    const profileImage = useMemo(() => (_jsx(Box, { width: "48px", minWidth: "48px", height: "48px", minHeight: "48px", borderRadius: '100%', bgcolor: profileColors[0], justifyContent: 'center', alignItems: 'center', sx: {
            display: "flex",
        }, children: _jsx(ProfileIcon, { fill: profileColors[1], width: '80%', height: '80%' }) })), [ProfileIcon, profileColors]);
    if (isOpen) {
        return (_jsxs(Stack, { direction: "column", children: [profileImage, isOpen && _jsx(Box, { width: "5px", height: "100%", borderRadius: '100px', bgcolor: profileColors[0], sx: {
                        marginLeft: "auto",
                        marginRight: "auto",
                        marginTop: 1,
                        marginBottom: 1,
                        cursor: "pointer",
                        "&:hover": {
                            brightness: palette.mode === "light" ? 1.05 : 0.95,
                        },
                    }, onClick: onToggle })] }));
    }
    return (_jsxs(Stack, { direction: "row", children: [_jsx(IconButton, { onClick: onToggle, sx: {
                    width: "48px",
                    height: "48px",
                }, children: _jsx(OpenThreadIcon, { fill: profileColors[0] }) }), profileImage] }));
};
//# sourceMappingURL=CommentConnector.js.map