/**
 * Drawer to display the steps of a routine, displayed as a vertical tree
 */
import { noop } from "@local/shared";
// import { TreeItem, treeItemClasses, TreeView } from "@mui/lab";
import { IconButton, Palette, SwipeableDrawer, useTheme } from "@mui/material";
import { useZIndex } from "hooks/useZIndex";
import { ListNumberIcon } from "icons";
import React, { useCallback, useMemo, useState } from "react";
import { addSearchParams, useLocation } from "route";
import { RunStepType } from "utils/consts";
import { RunStep, locationArraysMatch, routineVersionHasSubroutines } from "utils/runUtils";
import { MenuTitle } from "../MenuTitle/MenuTitle";
import { RunStepsDialogProps } from "../types";

interface StyledTreeItemProps {
    children?: React.ReactNode;
    label: string;
    isComplete: boolean;
    isSelected: boolean;
    nodeId: string;
    onLoad?: (ev: React.MouseEvent) => unknown;
    onToStep?: (ev: React.MouseEvent) => unknown;
    palette: Palette;
    type: RunStepType | "placeholder";
}

//TODO implement tree structure myself, since MUI TreeView not supported anymore
// const StyledTreeItem = styled((props: StyledTreeItemProps) => {
//     const {
//         label,
//         isComplete,
//         isSelected,
//         onLoad,
//         onToStep,
//         palette,
//         type,
//         ...other
//     } = props;

//     const canToStep = typeof onToStep === "function" && (type === RunStepType.Decision || type === RunStepType.Subroutine);

//     /**
//      * Only trigger onToStep if the onToStep function is supplied, and the type is a decision or subroutine. 
//      * Only trigger onLoad if the onLoad function is supplied.
//      */
//     const handleTreeItemClick = (ev: React.MouseEvent) => {
//         if (typeof onLoad === "function") {
//             onLoad(ev);
//         }
//         if (canToStep) {
//             onToStep(ev);
//         }
//     };

//     return (
//         <TreeItem
//             label={
//                 <Box
//                     onClick={handleTreeItemClick}
//                     sx={{
//                         display: "flex",
//                         alignItems: "center",
//                         p: 0.5,
//                         pr: 0,
//                         color: palette.background.textPrimary,
//                     }}
//                 >
//                     <Typography variant="body2" sx={{ fontWeight: "inherit", flexGrow: 1 }}>
//                         {label}
//                     </Typography>
//                     {/* Indicator for completeness */}
//                     {isComplete && <Checkbox
//                         id={`item-complete-checkbox-${props.nodeId}`}
//                         size="small"
//                         color='secondary'
//                         checked={true}
//                         disabled={true}
//                         sx={{
//                             "& svg": {
//                                 color: palette.secondary.main,
//                             },
//                         }}
//                     />}
//                     {/* Redirects to step */}
//                     {canToStep && <IconButton size="small" onClick={onToStep}>
//                         <OpenInNewIcon fill={palette.background.textPrimary} />
//                     </IconButton>}
//                 </Box>
//             }
//             {...other}
//         />
//     );
// })(({ theme }) => ({
//     [`& .${treeItemClasses.iconContainer}`]: {
//         "& .close": {
//             opacity: 0.3,
//         },
//     },
//     [`& .${treeItemClasses.group}`]: {
//         marginLeft: 15,
//         paddingLeft: 18,
//         borderLeft: `1px dashed ${alpha(theme.palette.text.primary, 0.4)}`,
//     },
//     [`& .${treeItemClasses.label}`]: {
//         fontSize: "1.2rem",
//         "& p": {
//             paddingTop: "12px",
//             paddingBottom: "12px",
//         },
//     },
// }));

const zIndexOffset = 1000;

export function RunStepsDialog({
    currStep,
    handleLoadSubroutine,
    handleCurrStepLocationUpdate,
    history,
    percentComplete,
    rootStep,
}: RunStepsDialogProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const [isOpen, setIsOpen] = useState(false);
    const zIndex = useZIndex(isOpen, false, zIndexOffset);
    const toggleOpen = useCallback(function toggleOpenCallback() {
        setIsOpen(!isOpen);
    }, [isOpen]);
    const closeDialog = useCallback(function closeDialogCallback() {
        setIsOpen(false);
    }, []);

    /**
     * Checks if a routine is complete. If it is a subroutine,
     * recursively checks all subroutine steps.
     * @param step current step
     * @param location Array indicating step location
     */
    const isComplete = useCallback((step: RunStep, location: number[]) => {
        switch (step.__type) {
            // If RoutineList, check all child steps
            case RunStepType.RoutineList:
            case RunStepType.Directory:
                return step.steps.every((childStep, index) => isComplete(childStep, [...location, index + 1]));
            // If Subroutine, check if subroutine is loaded
            case RunStepType.SingleRoutine:
                // Not loaded if subroutine has its own subroutines (since it would be converted to a RoutineList
                // if it was loaded)
                if (routineVersionHasSubroutines(step.routineVersion)) {
                    // We can't know if it is complete until it is loaded
                    return false;
                }
                // If simple routine
                return history.some(h => locationArraysMatch(h, location));
            // If decision
            case RunStepType.Decision:
                return history.some(h => locationArraysMatch(h, location));
        }
    }, [history]);

    const isSelected = useCallback((location: number[]) => {
        if (!currStep) return false;
        return locationArraysMatch(currStep, location);
    }, [currStep]);

    const selectedItem = useMemo<string | undefined>(() => {
        if (!currStep) return undefined;
        return currStep.join(".");
    }, [currStep]);

    /**
     * Generate a tree of the subroutine's steps
     */
    const getTreeItem = useCallback((step: RunStep, location: number[] = [1]) => {
        // Determine if step is completed/selected
        const completed = isComplete(step, location);
        const selected = isSelected(location);
        const locationLabel = location.join(".");
        // Helper function for navigating to step
        function toLocation() {
            // Update URL
            addSearchParams(setLocation, { step: location });
            // Update current step location
            console.log("updating current step location from dialog", location);
            handleCurrStepLocationUpdate(location);
            // Close dialog
            closeDialog();
        }
        // Item displayed differently depending on its type
        switch (step.__type) {
            // A decision step never has children
            case RunStepType.Decision:
                return null;
            // return <StyledTreeItem
            //     isComplete={completed}
            //     isSelected={selected}
            //     label={`${realLocationLabel}. Decision`}
            //     nodeId={locationLabel}
            //     onToStep={toLocation}
            //     palette={palette}
            //     type={step.type}
            // />;
            // A subroutine may have children, but they may not be loaded
            case RunStepType.SingleRoutine:
                // If there are further steps, add a "Loading" node
                if (routineVersionHasSubroutines(step.routineVersion)) {
                    return null;
                    // return (
                    //     <StyledTreeItem
                    //         isComplete={completed}
                    //         isSelected={selected}
                    //         label={`${realLocationLabel}. ${step.name}`}
                    //         nodeId={locationLabel}
                    //         onLoad={() => { handleLoadSubroutine(step.routineVersion.id); }} // Load subroutine(s)
                    //         onToStep={toLocation}
                    //         palette={palette}
                    //         type={step.type}
                    //     >
                    //         <StyledTreeItem
                    //             isComplete={false}
                    //             isSelected={false}
                    //             label="Loading..."
                    //             nodeId={`${locationLabel}-loading`}
                    //             palette={palette}
                    //             type={"placeholder"}
                    //         />
                    //     </StyledTreeItem>
                    // );
                }
                return null;
            // return <StyledTreeItem
            //     isComplete={completed}
            //     isSelected={selected}
            //     label={`${realLocationLabel}. ${step.name}`}
            //     nodeId={locationLabel}
            //     onToStep={toLocation}
            //     palette={palette}
            //     type={step.type}
            // />;

            // A routine list always has children
            case RunStepType.RoutineList:
            case RunStepType.Directory: {
                const stepItems = step.steps;
                // Don't wrap in a tree item if location is one element long (i.e. the root)
                if (location.length === 1) return stepItems.map((substep, i) => getTreeItem(substep, [...location, i + 1]));
                return null;
                // return (
                //     <StyledTreeItem
                //         isComplete={completed}
                //         isSelected={selected}
                //         label={`${realLocationLabel}. ${step.name}`}
                //         nodeId={locationLabel}
                //         palette={palette}
                //         type={step.type}
                //     >
                //         {stepItems.map((substep, i) => getTreeItem(substep, [...location, i + 1]))}
                //     </StyledTreeItem>
                // );
            }
        }
    }, [isComplete, isSelected, setLocation, handleCurrStepLocationUpdate, closeDialog]);

    return (
        <>
            {/* Icon for opening/closing dialog */}
            <IconButton edge="start" color="inherit" aria-label="menu" onClick={toggleOpen}>
                <ListNumberIcon width='32px' height='32px' />
            </IconButton>
            {/* The dialog */}
            <SwipeableDrawer
                anchor="right"
                open={isOpen}
                onOpen={noop}
                onClose={closeDialog}
                sx={{
                    zIndex,
                    "& .MuiDrawer-paper": {
                        background: palette.background.default,
                        minHeight: "100vh",
                        minWidth: "300px",
                        overflowY: "auto",
                    },
                }}
            >
                <MenuTitle
                    onClose={closeDialog}
                    title={`Steps (${Math.floor(percentComplete)}% Complete)`}
                />
                {/* Tree display of steps */}
                {/* <TreeView
                    aria-label="routine steps navigator"
                    defaultCollapseIcon={<StepListCloseIcon fill={palette.background.textPrimary} />}
                    defaultExpandIcon={<StepListOpenIcon fill={palette.background.textPrimary} />}
                    defaultEndIcon={<StepListEndIcon fill={palette.background.textSecondary} />}
                    selected={selectedItem}
                    sx={{
                        height: 240,
                        flexGrow: 1,
                        maxWidth: 400,
                        overflowY: "auto",
                        // Add padding to bottom to account for iOS navbar (safe-area-inset-bottom not working for some reason)
                        paddingBottom: "80px",
                    }}
                >
                    {rootStep && getTreeItem(rootStep)}
                </TreeView> */}
            </SwipeableDrawer>
        </>
    );
}
