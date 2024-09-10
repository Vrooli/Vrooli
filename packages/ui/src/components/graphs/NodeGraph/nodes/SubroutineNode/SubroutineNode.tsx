import { Routine } from "@local/shared";
import { Box, BoxProps, IconButton, Tooltip, Typography, TypographyProps, styled, useTheme } from "@mui/material";
import { usePress } from "hooks/gestures";
import { ActionIcon, CloseIcon, NoActionIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { multiLineEllipsis, noSelect } from "styles";
import { BuildAction } from "utils/consts";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { routineTypes } from "utils/search/schemas/routine";
import { SHOW_TITLE_ABOVE_SCALE, calculateNodeSize } from "..";
import { NodeWidth } from "../..";
import { NodeContextMenu } from "../../NodeContextMenu/NodeContextMenu";
import { routineNodeActionStyle, routineNodeCheckboxOption } from "../styles";
import { SubroutineNodeProps } from "../types";

/**
 * Decides if a clicked element should trigger opening the subroutine dialog 
 * @param id ID of the clicked element
 */
function shouldOpen(id: string | null | undefined): boolean {
    // Only collapse if clicked on name bar or name
    return Boolean(id && (id.startsWith("subroutine-name-")));
}

interface OuterBoxProps extends BoxProps {
    nodeSize: string;
}

const OuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "nodeSize",
})<OuterBoxProps>(({ nodeSize, theme }) => ({
    boxShadow: theme.shadows[2],
    minWidth: nodeSize,
    position: "relative",
    display: "block",
    marginBottom: "8px",
    borderRadius: "12px",
    overflow: "hidden",
    background: theme.palette.mode === "light" ? "#b0bbe7" : "#384164",
    color: theme.palette.background.textPrimary,
    "@media print": {
        border: `1px solid ${theme.palette.mode === "light" ? "#b0bbe7" : "#384164"}`,
        boxShadow: "none",
    },
}));

interface InnerBoxProps extends BoxProps {
    canUpdate: boolean;
}

const InnerBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "canUpdate",
})<InnerBoxProps>(({ canUpdate, theme }) => ({
    display: "flex",
    alignItems: "center",
    background: canUpdate ?
        (theme.palette.mode === "light" ? theme.palette.primary.dark : theme.palette.secondary.dark) :
        "#667899",
    color: theme.palette.mode === "light" ? theme.palette.primary.contrastText : theme.palette.secondary.contrastText,
    padding: "0.1em",
    paddingLeft: "8px!important",
    paddingRight: "8px!important",
    textAlign: "center",
    cursor: "pointer",
    "&:hover": {
        filter: "brightness(120%)",
        transition: "filter 0.2s",
    },
}));

interface NodeTitleProps extends TypographyProps {
    fontSize: string;
    scale: number;
}

const NodeTitle = styled(Typography, {
    shouldForwardProp: (prop) => prop !== "fontSize" && prop !== "scale",
})<NodeTitleProps>(({ fontSize, scale }) => ({
    ...noSelect,
    ...multiLineEllipsis(2),
    fontSize,
    textAlign: "center",
    width: "100%",
    lineBreak: scale < SHOW_TITLE_ABOVE_SCALE ? "anywhere" : "normal",
    display: scale < SHOW_TITLE_ABOVE_SCALE ? "none" : "block",
    "@media print": {
        color: "black",
    },
}));

const isOptionalButtonStyle = { ...routineNodeCheckboxOption };

export function SubroutineNode({
    data,
    scale = 1,
    labelVisible,
    isEditing,
    handleAction,
    handleUpdate,
}: SubroutineNodeProps) {
    const { palette } = useTheme();

    const nodeSize = useMemo(() => `${calculateNodeSize(220, scale)}px`, [scale]);
    const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.RoutineList, scale) / 9}px, 1.5em)`, [scale]);
    // Determines if the subroutine is one you can edit
    const canUpdate = useMemo<boolean>(() => ((data?.routineVersion?.root as Routine)?.isInternal ?? (data?.routineVersion?.root as Routine)?.you?.canUpdate === true), [data.routineVersion]);

    const { title } = useMemo(() => getDisplay({ ...data, __typename: "NodeRoutineListItem" }, navigator.languages), [data]);

    const onAction = useCallback((event: any | null, action: BuildAction.OpenSubroutine | BuildAction.EditSubroutine | BuildAction.DeleteSubroutine) => {
        if (event && [BuildAction.EditSubroutine, BuildAction.DeleteSubroutine].includes(action)) {
            event.stopPropagation();
        }
        handleAction(action, data.id);
    }, [data.id, handleAction]);
    const openSubroutine = useCallback((target: EventTarget) => {
        if (!shouldOpen(target.id)) return;
        onAction(null, BuildAction.OpenSubroutine);
    }, [onAction]);
    const deleteSubroutine = useCallback((event: any) => { onAction(event, BuildAction.DeleteSubroutine); }, [onAction]);

    const toggleIsOptional = useCallback(function toggleIsOptionalCallback() {
        if (!isEditing) return;
        handleUpdate(data.id, {
            ...data,
            isOptional: !data.isOptional,
        });
    }, [isEditing, handleUpdate, data]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `subroutine-context-menu-${data.id}`, [data?.id]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target: EventTarget) => {
        // Ignore if not editing
        if (!isEditing) return;
        setContextAnchor(target);
    }, [isEditing]);
    const closeContext = useCallback(function closeContextCallback() { setContextAnchor(null); }, []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onClick: openSubroutine,
        onRightClick: openContext,
    });

    const availableActions = useMemo(() => {
        return isEditing ?
            [BuildAction.EditSubroutine, BuildAction.DeleteSubroutine] :
            [BuildAction.OpenSubroutine];
    }, [isEditing]);
    const handleContextSelect = useCallback((action: BuildAction) => { onAction(null, action as BuildAction.EditSubroutine | BuildAction.DeleteSubroutine | BuildAction.OpenSubroutine); }, [onAction]);

    const RoutineTypeIcon = useMemo(() => {
        const routineType = data?.routineVersion?.routineType;
        if (!routineType || !routineTypes[routineType]) return null;
        return routineTypes[routineType].icon;
    }, [data]);

    return (
        <>
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={availableActions}
                handleClose={closeContext}
                handleSelect={handleContextSelect}
            />
            <OuterBox nodeSize={nodeSize}>
                <InnerBox
                    canUpdate={canUpdate}
                    id={`subroutine-name-bar-${data.id}`}
                    {...pressEvents}
                    aria-owns={contextOpen ? contextId : undefined}
                >
                    <Tooltip placement={"top"} title='Routine can be skipped'>
                        <Box
                            onClick={toggleIsOptional}
                            onTouchStart={toggleIsOptional}
                            sx={routineNodeActionStyle(isEditing)}
                        >
                            <IconButton
                                size="small"
                                sx={isOptionalButtonStyle}
                                disabled={!isEditing}
                            >
                                {data?.isOptional ? <NoActionIcon /> : <ActionIcon />}
                            </IconButton>
                        </Box>
                    </Tooltip>
                    {RoutineTypeIcon && <RoutineTypeIcon width={16} height={16} fill={palette.background.textPrimary} />}
                    {labelVisible && <NodeTitle
                        fontSize={fontSize}
                        id={`subroutine-name-${data.id}`}
                        scale={scale}
                        variant="h6"
                    >{firstString(title, "Untitled")}</NodeTitle>}
                    {isEditing && (
                        <IconButton
                            id={`subroutine-delete-icon-button-${data.id}`}
                            onClick={deleteSubroutine}
                            onTouchStart={deleteSubroutine}
                            color="inherit"
                        >
                            <CloseIcon id={`subroutine-delete-icon-${data.id}`} />
                        </IconButton>
                    )}
                </InnerBox>
            </OuterBox>
        </>
    );
}
