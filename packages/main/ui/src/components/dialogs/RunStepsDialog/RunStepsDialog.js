import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { ListNumberIcon, OpenInNewIcon, StepListClose, StepListEnd, StepListOpen } from "@local/icons";
import { TreeItem, treeItemClasses, TreeView } from "@mui/lab";
import { alpha, Box, Checkbox, IconButton, styled, SwipeableDrawer, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { RoutineStepType } from "../../../utils/consts";
import { addSearchParams, useLocation } from "../../../utils/route";
import { locationArraysMatch, routineVersionHasSubroutines } from "../../../utils/runUtils";
import { MenuTitle } from "../MenuTitle/MenuTitle";
const StyledTreeItem = styled((props) => {
    const { label, isComplete, isSelected, onLoad, onToStep, palette, type, ...other } = props;
    const canToStep = typeof onToStep === "function" && (type === RoutineStepType.Decision || type === RoutineStepType.Subroutine);
    const handleTreeItemClick = (ev) => {
        if (typeof onLoad === "function") {
            onLoad(ev);
        }
        if (canToStep) {
            onToStep(ev);
        }
    };
    return (_jsx(TreeItem, { label: _jsxs(Box, { onClick: handleTreeItemClick, sx: {
                display: "flex",
                alignItems: "center",
                p: 0.5,
                pr: 0,
                color: palette.background.textPrimary,
            }, children: [_jsx(Typography, { variant: "body2", sx: { fontWeight: "inherit", flexGrow: 1 }, children: label }), isComplete && _jsx(Checkbox, { id: `item-complete-checkbox-${props.nodeId}`, size: "small", color: 'secondary', checked: true, disabled: true, sx: {
                        "& svg": {
                            color: palette.secondary.main,
                        },
                    } }), canToStep && _jsx(IconButton, { size: "small", onClick: onToStep, children: _jsx(OpenInNewIcon, { fill: palette.background.textPrimary }) })] }), ...other }));
})(({ theme }) => ({
    [`& .${treeItemClasses.iconContainer}`]: {
        "& .close": {
            opacity: 0.3,
        },
    },
    [`& .${treeItemClasses.group}`]: {
        marginLeft: 15,
        paddingLeft: 18,
        borderLeft: `1px dashed ${alpha(theme.palette.text.primary, 0.4)}`,
    },
    [`& .${treeItemClasses.label}`]: {
        fontSize: "1.2rem",
        "& p": {
            paddingTop: "12px",
            paddingBottom: "12px",
        },
    },
}));
export const RunStepsDialog = ({ currStep, handleLoadSubroutine, handleCurrStepLocationUpdate, history, percentComplete, stepList, zIndex, }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const [isOpen, setIsOpen] = useState(false);
    const toggleOpen = useCallback(() => setIsOpen(!isOpen), [isOpen]);
    const closeDialog = () => { setIsOpen(false); };
    const isComplete = useCallback((step, location) => {
        switch (step.type) {
            case RoutineStepType.RoutineList:
                return step.steps.every((childStep, index) => isComplete(childStep, [...location, index + 1]));
            case RoutineStepType.Subroutine:
                if (routineVersionHasSubroutines(step.routineVersion)) {
                    return false;
                }
                return history.some(h => locationArraysMatch(h, location));
            case RoutineStepType.Decision:
                return history.some(h => locationArraysMatch(h, location));
        }
    }, [history]);
    const isSelected = useCallback((location) => {
        if (!currStep)
            return false;
        return locationArraysMatch(currStep, location);
    }, [currStep]);
    const selectedItem = useMemo(() => {
        if (!currStep)
            return undefined;
        return `1.${currStep.join(".")}`;
    }, [currStep]);
    const getTreeItem = useCallback((step, location = [1]) => {
        const realLocation = location.slice(1);
        const completed = isComplete(step, realLocation);
        const selected = isSelected(realLocation);
        const locationLabel = location.join(".");
        const realLocationLabel = realLocation.join(".");
        const toLocation = () => {
            addSearchParams(setLocation, { step: realLocation });
            handleCurrStepLocationUpdate(realLocation);
            closeDialog();
        };
        switch (step.type) {
            case RoutineStepType.Decision:
                return _jsx(StyledTreeItem, { isComplete: completed, isSelected: selected, label: `${realLocationLabel}. Decision`, nodeId: locationLabel, onToStep: toLocation, palette: palette, type: step.type });
            case RoutineStepType.Subroutine:
                if (routineVersionHasSubroutines(step.routineVersion)) {
                    return (_jsx(StyledTreeItem, { isComplete: completed, isSelected: selected, label: `${realLocationLabel}. ${step.name}`, nodeId: locationLabel, onLoad: () => { handleLoadSubroutine(step.routineVersion.id); }, onToStep: toLocation, palette: palette, type: step.type, children: _jsx(StyledTreeItem, { isComplete: false, isSelected: false, label: "Loading...", nodeId: `${locationLabel}-loading`, palette: palette, type: "placeholder" }) }));
                }
                return _jsx(StyledTreeItem, { isComplete: completed, isSelected: selected, label: `${realLocationLabel}. ${step.name}`, nodeId: locationLabel, onToStep: toLocation, palette: palette, type: step.type });
            case RoutineStepType.RoutineList:
                const stepItems = step.steps;
                if (location.length === 1)
                    return stepItems.map((substep, i) => getTreeItem(substep, [...location, i + 1]));
                return (_jsx(StyledTreeItem, { isComplete: completed, isSelected: selected, label: `${realLocationLabel}. ${step.name}`, nodeId: locationLabel, palette: palette, type: step.type, children: stepItems.map((substep, i) => getTreeItem(substep, [...location, i + 1])) }));
        }
    }, [isComplete, isSelected, setLocation, handleCurrStepLocationUpdate, palette, handleLoadSubroutine]);
    return (_jsxs(_Fragment, { children: [_jsx(IconButton, { edge: "start", color: "inherit", "aria-label": "menu", onClick: toggleOpen, children: _jsx(ListNumberIcon, { width: '32px', height: '32px' }) }), _jsxs(SwipeableDrawer, { anchor: "right", open: isOpen, onOpen: () => { }, onClose: closeDialog, sx: {
                    zIndex,
                    "& .MuiDrawer-paper": {
                        background: palette.background.default,
                        minHeight: "100vh",
                        minWidth: "300px",
                        overflowY: "auto",
                    },
                }, children: [_jsx(MenuTitle, { onClose: closeDialog, title: `Steps (${Math.floor(percentComplete)}% Complete)` }), _jsx(TreeView, { "aria-label": "routine steps navigator", defaultCollapseIcon: _jsx(StepListClose, { fill: palette.background.textPrimary }), defaultExpandIcon: _jsx(StepListOpen, { fill: palette.background.textPrimary }), defaultEndIcon: _jsx(StepListEnd, { fill: palette.background.textSecondary }), selected: selectedItem, sx: {
                            height: 240,
                            flexGrow: 1,
                            maxWidth: 400,
                            overflowY: "auto",
                            paddingBottom: "80px",
                        }, children: stepList && getTreeItem(stepList) })] })] }));
};
//# sourceMappingURL=RunStepsDialog.js.map