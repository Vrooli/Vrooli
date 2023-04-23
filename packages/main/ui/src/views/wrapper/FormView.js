import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Container, useTheme } from "@mui/material";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
export const FormView = ({ display = "page", title, autocomplete = "on", children, maxWidth = "90%", }) => {
    const { palette } = useTheme();
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    title,
                } }), _jsx(Box, { sx: {
                    backgroundColor: palette.background.paper,
                    display: "grid",
                    position: "relative",
                    boxShadow: "0px 2px 4px -1px rgb(0 0 0 / 20%), 0px 4px 5px 0px rgb(0 0 0 / 14%), 0px 1px 10px 0px rgb(0 0 0 / 12%)",
                    minWidth: "300px",
                    maxWidth: "min(100%, 700px)",
                    borderRadius: "10px",
                    overflow: "hidden",
                    left: "50%",
                    transform: "translateX(-50%)",
                    marginBottom: "20px",
                }, children: _jsx(Container, { children: children }) })] }));
};
//# sourceMappingURL=FormView.js.map