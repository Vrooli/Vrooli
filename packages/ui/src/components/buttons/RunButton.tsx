import { HistoryPageTabOption, LINKS, ListObject, ProjectVersionTranslation, RoutineVersionTranslation, RunProject, RunRoutine, RunStatus, RunViewSearchParams, Status, camelCase, funcFalse, getTranslation, noop } from "@local/shared";
import { Box, Button, IconButton, Menu, Tooltip, styled, useTheme } from "@mui/material";
import React, { useCallback, useContext, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { useUpsertRunProject, useUpsertRunRoutine } from "../../hooks/runs.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { usePopover } from "../../hooks/usePopover.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { ArgsType } from "../../types.js";
import { getDummyListLength } from "../../utils/consts.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { PubSub } from "../../utils/pubsub.js";
import { createRunPath } from "../../views/runs/RunView.js";
import { ListContainer } from "../containers/ListContainer.js";
import { MenuTitle } from "../dialogs/MenuTitle/MenuTitle.js";
import { PopoverWithArrow } from "../dialogs/PopoverWithArrow/PopoverWithArrow.js";
import { ObjectList } from "../lists/ObjectList/ObjectList.js";
import { ObjectListActions } from "../lists/types.js";
import { RunButtonProps } from "./types.js";

const emptyArray = [];

const titleId = "run-picker-dialog-title";
const anchorOrigin = {
    vertical: "bottom",
    horizontal: "center",
} as const;
const transformOrigin = {
    vertical: "top",
    horizontal: "center",
} as const;

export interface RunPickerMenuProps {
    anchorEl: Element | null;
    handleClose: () => unknown;
    objectId: string | null;
    objectName: string | null;
    objectType: "ProjectVersion" | "RoutineVersion";
    onSelect: (run: RunProject | RunRoutine | null) => unknown;
}

const RunMenu = styled(Menu)(({ theme }) => ({
    "& .MuiPaper-root": {
        position: "absolute",
        bottom: 0,
        right: 0,
        top: "unset!important",
        left: "unset!important",
        maxHeight: "calc(100vh - 32px)",
    },
    "& .MuiMenu-paper": {
        background: theme.palette.background.default,
    },
    "& .MuiMenu-list": {
        paddingTop: 0,
        paddingBottom: 0,
    },
}));
const NewRunButton = styled(Button)(({ theme }) => ({
    width: "-webkit-fill-available",
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
    marginLeft: theme.spacing(2),
    marginRight: theme.spacing(2),
}));

/**
 * Handles selecting a run from a list of runs.
 */
export function RunPickerMenu({
    anchorEl,
    handleClose,
    objectId,
    objectName,
    objectType,
    onSelect,
}: RunPickerMenuProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const open = useMemo(() => Boolean(anchorEl), [anchorEl]);

    const lastRunSelected = useRef<string | null>(null);
    const where = useMemo(function whereMemo() {
        const result: object = {};
        if (objectId) {
            result[`${camelCase(objectType)}Id`] = objectId;
        }
        if (Object.keys(result).length === 0) return undefined;
        result["status"] = RunStatus.InProgress;
        return result;
    }, [objectId, objectType]);

    const { allData, loading, removeItem, setAllData, updateItem } = useFindMany<ListObject>({
        canSearch: () => typeof where === "object",
        controlsUrl: false,
        searchType: objectType === "ProjectVersion" ? "RunProject" : "RunRoutine",
        take: 10,
        where,
    });

    const { createRun: createRunProject } = useUpsertRunProject();
    const { createRun: createRunRoutine } = useUpsertRunRoutine();

    const createNewRun = useCallback(() => {
        if (!objectId) {
            PubSub.get().publish("snack", { message: t("CouldNotReadObject", { ns: "error" }), severity: "Error" });
            return;
        }

        function onSuccess(data: RunProject | RunRoutine) {
            lastRunSelected.current = data.id;
            setAllData(list => [data, ...list]);
            onSelect(data);
            handleClose();
        }

        if (objectType === "ProjectVersion") {
            createRunProject({ objectId, objectName, onSuccess });
        }
        else {
            createRunRoutine({ objectId, objectName, onSuccess });
        }
    }, [handleClose, onSelect, setAllData, createRunProject, createRunRoutine, objectId, objectName, objectType, t]);

    useEffect(function checkLoggedIn() {
        if (!open) return;
        // If not logged in, open without creating a new run
        if (session?.isLoggedIn !== true) {
            console.warn("User not logged in. Cannot create new run");
            console.log("session", session);
            onSelect(null);
            handleClose();
        }
    }, [open, createNewRun, onSelect, session, handleClose]);

    const handleOpen = useCallback(function selectItem(data: ListObject) {
        onSelect(data as RunProject | RunRoutine);
        handleClose();
    }, [handleClose, onSelect]);

    const onAction = useCallback((action: keyof ObjectListActions<ListObject>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted":
                removeItem(...(data as ArgsType<ObjectListActions<ListObject>["Deleted"]>));
                break;
            case "Updated":
                updateItem(...(data as ArgsType<ObjectListActions<ListObject>["Updated"]>));
                break;
        }
    }, [removeItem, updateItem]);

    return (
        <RunMenu
            id='select-run-dialog'
            aria-labelledby={titleId}
            disableScrollLock={true}
            open={open}
            anchorEl={anchorEl}
            anchorOrigin={anchorOrigin}
            transformOrigin={transformOrigin}
            onClose={handleClose}
        >
            <Box display="flex" flexDirection="column" height="100%" maxHeight="calc(100vh - 40px)">
                <MenuTitle
                    ariaLabel={titleId}
                    onClose={handleClose}
                    title={"Continue Existing Run?"}
                />
                <Box flexGrow={1} overflow="auto">
                    <ListContainer
                        borderRadius={0}
                        emptyText={"No existing runs found"}
                        isEmpty={allData.length === 0 && !loading}
                    >
                        <ObjectList
                            canNavigate={funcFalse}
                            dummyItems={new Array(getDummyListLength("dialog")).fill("RunRoutine")}
                            handleToggleSelect={noop}
                            hideUpdateButton={true}
                            isSelecting={false}
                            items={allData}
                            keyPrefix={"run-list-item"}
                            loading={loading}
                            onAction={onAction}
                            onClick={handleOpen}
                            selectedItems={emptyArray}
                        />
                    </ListContainer>
                    {allData.length > 0 && <Box display="flex" alignItems="center" justifyContent="flex-end">
                        <Button
                            endIcon={<IconCommon decorative name="ArrowRight" />}
                            href={`${LINKS.History}?type="${HistoryPageTabOption.RunsActive}"&routineVersionId="${objectId}"`}
                            variant="text"
                        >
                            {t("More")}
                        </Button>
                    </Box>}
                </Box>
                <Box>
                    <NewRunButton
                        color="secondary"
                        onClick={createNewRun}
                        variant="contained"
                    >New Run</NewRunButton>
                </Box>
            </Box>
        </RunMenu>
    );
}

/**
 * Button to run a multi-step routine. 
 * If the routine is invalid, it is greyed out with a tooltip on hover or press. 
 * If the routine is incomplete, the button is available but the user must confirm an alert before running.
 */
export function RunButton({
    isEditing,
    objectType,
    runnableObject,
}: RunButtonProps) {
    const session = useContext(SessionContext);
    // const languages = useMemo(() => getUserLanguages(session), [session]);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const status = useMemo<Status>(function statusMemo() {
        if (!runnableObject) return Status.Invalid;
        // Add logic to check if the routine is valid if needed. 
        // Othwerwise, we'll just let users try to run it and see what happens.
        return Status.Valid;
    }, [runnableObject]);

    const [selectRunAnchor, openSelectRunDialog, closeSelectRunDialog] = usePopover();
    const handleRunSelect = useCallback((run: RunProject | RunRoutine | null) => {
        if (!runnableObject) return;
        // If run is null, it means the routine will be opened without a run
        if (!run) {
            const searchParams: RunViewSearchParams = { step: [1] };
            const path = createRunPath("test", runnableObject);
            setLocation(path, { searchParams });
        }
        // Otherwise, open routine where last left off in run
        else {
            const searchParams: RunViewSearchParams = { step: run.lastStep ?? undefined };
            const path = createRunPath(run.id, runnableObject);
            setLocation(path, { searchParams });
        }
    }, [runnableObject, setLocation]);

    const startRun = useCallback((event: React.MouseEvent<HTMLElement>) => {
        if (!runnableObject) return;
        // If editing, don't use a real run
        if (isEditing) {
            const searchParams: RunViewSearchParams = { step: [1] };
            const path = createRunPath("test", runnableObject);
            setLocation(path, { searchParams });
        }
        else {
            openSelectRunDialog(event);
        }
    }, [isEditing, openSelectRunDialog, runnableObject, setLocation]);

    // Invalid message popup
    const [errorAnchorEl, openErrorPopup, closeErrorPopup] = usePopover();
    const openError = useCallback((ev: React.MouseEvent<HTMLElement>) => {
        ev.preventDefault();
        openErrorPopup(ev);
    }, [openErrorPopup]);

    const runStart = useCallback((event: React.MouseEvent<HTMLElement>) => {
        // If invalid, don't run
        if (status === Status.Invalid) {
            openError(event);
            return;
        }
        // If incomplete, confirm user wants to run
        else if (status === Status.Incomplete) {
            PubSub.get().publish("alertDialog", {
                messageKey: "RunInvalidRoutineConfirm",
                buttons: [
                    { labelKey: "Yes", onClick: () => { startRun(event); } },
                    { labelKey: "Cancel" },
                ],
            });
        }
        // Otherwise, run
        else {
            startRun(event);
        }
    }, [openError, startRun, status]);

    return (
        <>
            {/* Invalid routine popup */}
            <PopoverWithArrow
                anchorEl={errorAnchorEl}
                handleClose={closeErrorPopup}
            >{t("RoutineCannotRunInvalid", { ns: "error" })}</PopoverWithArrow>
            {/* Chooses which run to use */}
            <RunPickerMenu
                anchorEl={selectRunAnchor}
                handleClose={closeSelectRunDialog}
                objectId={runnableObject?.id ?? null}
                objectName={getTranslation<ProjectVersionTranslation | RoutineVersionTranslation>(runnableObject, getUserLanguages(session)).name ?? null}
                objectType={objectType}
                onSelect={handleRunSelect}
            />
            {/* Run button */}
            <Tooltip title="Run Routine" placement="top">
                {/* Button wrapped in div so it can be pressed when disabled */}
                <Box onClick={runStart}>
                    <IconButton
                        disabled={status === Status.Invalid}
                    >
                        <IconCommon name="Play" />
                    </IconButton>
                </Box>
            </Tooltip>
        </>
    );
}
