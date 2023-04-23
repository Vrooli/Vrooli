import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { NodeType } from "@local/consts";
import { OpenInNewIcon } from "@local/icons";
import { ListItem, ListItemButton, ListItemText, Stack, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { multiLineEllipsis } from "../../../styles";
import { getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { SessionContext } from "../../../utils/SessionContext";
export const DecisionView = ({ data, display = "page", handleDecisionSelect, nodes, zIndex, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const decisions = useMemo(() => {
        return data.links.map(link => {
            const node = nodes.find(n => n.id === link.to.id);
            let color = palette.primary.dark;
            if (node?.nodeType === NodeType.End) {
                color = node.end?.wasSuccessful === false ? "#7c262a" : "#387e30";
            }
            return { node, link, color };
        });
    }, [data.links, nodes, palette.primary.dark]);
    const toDecision = useCallback((index) => {
        const decision = decisions[index];
        handleDecisionSelect(decision.node);
    }, [decisions, handleDecisionSelect]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Decision",
                    helpKey: "DecisionHelp",
                } }), _jsxs(Stack, { direction: "column", spacing: 4, p: 2, children: [_jsx(Stack, { direction: "row", justifyContent: "center", alignItems: "center", children: _jsx(Typography, { variant: "h4", sx: { textAlign: "center" }, children: "What would you like to do next?" }) }), decisions.map((decision, index) => {
                        const languages = getUserLanguages(session);
                        const { description, name } = getTranslation(decision.node, languages, true);
                        return (_jsx(ListItem, { disablePadding: true, onClick: () => { toDecision(index); }, sx: {
                                display: "flex",
                                background: decision.color,
                                color: "white",
                                boxShadow: 12,
                                borderRadius: "12px",
                            }, children: _jsxs(ListItemButton, { component: "div", onClick: () => { toDecision(index); }, children: [_jsxs(Stack, { direction: "column", spacing: 1, pl: 2, sx: { width: "-webkit-fill-available", alignItems: "center" }, children: [_jsx(ListItemText, { primary: name, sx: {
                                                    ...multiLineEllipsis(1),
                                                    fontWeight: "bold",
                                                } }), description && _jsx(ListItemText, { primary: description, sx: {
                                                    ...multiLineEllipsis(2),
                                                } })] }), _jsx(OpenInNewIcon, { fill: "white" })] }) }));
                    })] })] }));
};
//# sourceMappingURL=DecisionView.js.map