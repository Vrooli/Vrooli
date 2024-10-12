import { HistoryPageTabOption, LINKS, ListObject, ProjectVersion, ProjectVersionTranslation, RoutineVersion, RoutineVersionTranslation, RunProject, RunProjectCreateInput, RunRoutine, RunRoutineCreateInput, RunStatus, Status, base36ToUuid, camelCase, endpointPostRunProject, endpointPostRunRoutine, funcFalse, getTranslation, isOfType, noop, parseSearchParams, projectVersionStatus, routineVersionStatus, uuid, uuidToBase36, uuidValidate } from "@local/shared";
import { Box, Button, Menu, Tooltip, styled, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { MenuTitle } from "components/dialogs/MenuTitle/MenuTitle";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ObjectListActions } from "components/lists/types";
import { SessionContext } from "contexts";
import { useFindMany } from "hooks/useFindMany";
import { useLazyFetch } from "hooks/useLazyFetch";
import { usePopover } from "hooks/usePopover";
import { ArrowRightIcon, PlayIcon } from "icons";
import React, { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { setSearchParams, useLocation } from "route";
import { SideActionsButton } from "styles";
import { ArgsType } from "types";
import { getDummyListLength } from "utils/consts";
import { getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { RunView } from "views/runs/RunView/RunView";
import { RunButtonProps } from "../types";

const emptyArray = [];

function getRunIdFromUrl(): string | null {
    const searchParams = parseSearchParams();
    if (!searchParams.run || typeof searchParams.run !== "string") return null;
    const runId = searchParams.run === "test" ? "test" : base36ToUuid(searchParams.run);
    if (runId === "test" || uuidValidate(runId)) return runId;
    return null;
}

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
    anchorEl: HTMLElement | null;
    handleClose: () => unknown;
    objectId: string | null;
    objectName: string | null;
    objectType: "ProjectVersion" | "RoutineVersion";
    onSelect: (run: RunProject | RunRoutine | null) => unknown;
    runnableObject?: Partial<RoutineVersion | ProjectVersion> | null;
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
        const runId = getRunIdFromUrl();
        if (runId) {
            result["ids"] = [runId];
        }
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

    // If runId is in the URL, select that run automatically
    useEffect(function selectRunFromUrl() {
        if (allData.length === 0) return;
        const runId = getRunIdFromUrl();
        const run = allData.find(run => run.id === runId);
        if (run && run.id !== lastRunSelected.current) {
            lastRunSelected.current = run.id ?? null;
            onSelect(run as RunProject | RunRoutine);
            handleClose();
        }
    }, [allData, onSelect, handleClose]);

    const [createRunProject] = useLazyFetch<RunProjectCreateInput, RunProject>(endpointPostRunProject);
    const [createRunRoutine] = useLazyFetch<RunRoutineCreateInput, RunRoutine>(endpointPostRunRoutine);
    const createNewRun = useCallback(() => {
        if (!objectId) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }

        function handleSuccess(data: RunProject | RunRoutine) {
            lastRunSelected.current = data.id;
            setAllData(list => [data, ...list]);
            onSelect(data);
            handleClose();
        }

        if (objectType === "ProjectVersion") {
            fetchLazyWrapper<RunProjectCreateInput, RunProject>({
                fetch: createRunProject,
                inputs: {
                    id: uuid(),
                    isPrivate: true,
                    name: objectName ?? "Unnamed Project",
                    projectVersionConnect: objectId,
                    status: RunStatus.InProgress,
                },
                successCondition: (data) => data !== null,
                onSuccess: handleSuccess,
                errorMessage: () => ({ messageKey: "FailedToCreateRun" }),
            });
        }
        else {
            fetchLazyWrapper<RunRoutineCreateInput, RunRoutine>({
                fetch: createRunRoutine,
                inputs: {
                    id: uuid(),
                    isPrivate: true,
                    name: objectName ?? "Unnamed Routine",
                    routineVersionConnect: objectId,
                    status: RunStatus.InProgress,
                },
                successCondition: (data) => data !== null,
                onSuccess: handleSuccess,
                errorMessage: () => ({ messageKey: "FailedToCreateRun" }),
            });
        }
    }, [handleClose, onSelect, setAllData, createRunProject, createRunRoutine, objectId, objectName, objectType]);

    useEffect(() => {
        if (!open) return;
        // If not logged in, open without creating a new run
        if (session?.isLoggedIn !== true) {
            onSelect(null);
            handleClose();
        }
    }, [open, createNewRun, onSelect, session?.isLoggedIn, handleClose]);

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
                            endIcon={<ArrowRightIcon />}
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
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const status = useMemo<Status>(function statusMemo() {
        if (!runnableObject) return Status.Invalid;
        if (isOfType(runnableObject, "ProjectVersion")) {
            return projectVersionStatus(runnableObject).status;
        }
        if (isOfType(runnableObject, "RoutineVersion")) {
            const routineType = runnableObject.routineType;
            if (!routineType) return Status.Invalid;
            return routineVersionStatus(routineType, runnableObject).status;
        }
        return Status.Invalid;
    }, [runnableObject]);

    const [isRunOpen, setIsRunOpen] = useState(() => {
        const params = parseSearchParams();
        return typeof params.run === "string" && uuidValidate(params.run);
    });
    const [selectRunAnchor, openSelectRunDialog, closeSelectRunDialog] = usePopover();
    const handleRunSelect = useCallback((run: RunProject | RunRoutine | null) => {
        // If run is null, it means the routine will be opened without a run
        if (!run) {
            setSearchParams(setLocation, {
                run: "test",
                step: [1],
            });
        }
        // Otherwise, open routine where last left off in run
        else {
            setSearchParams(setLocation, {
                run: uuidToBase36(run.id),
                step: run.lastStep ?? undefined,
            });
        }
        setIsRunOpen(true);
    }, [setLocation]);

    const startRun = useCallback((event: React.MouseEvent<HTMLElement>) => {
        // If editing, don't use a real run
        if (isEditing) {
            setSearchParams(setLocation, {
                run: "test",
                step: [1],
            });
            setIsRunOpen(true);
        }
        else {
            openSelectRunDialog(event);
        }
    }, [isEditing, openSelectRunDialog, setLocation]);

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

    const runStop = useCallback(function runStopCallback() {
        setLocation(window.location.pathname, { replace: true });
        setIsRunOpen(false);
    }, [setLocation]);

    return (
        <>
            {/* Invalid routine popup */}
            <PopoverWithArrow
                anchorEl={errorAnchorEl}
                handleClose={closeErrorPopup}
            >{t("RoutineCannotRunInvalid", { ns: "error" })}</PopoverWithArrow>
            {/* Run dialog */}
            {runnableObject && <RunView
                display="dialog"
                isOpen={isRunOpen}
                onClose={runStop}
                runnableObject={runnableObject}
            />}
            {/* Chooses which run to use */}
            <RunPickerMenu
                anchorEl={selectRunAnchor}
                handleClose={closeSelectRunDialog}
                objectId={runnableObject?.id ?? null}
                objectName={getTranslation<ProjectVersionTranslation | RoutineVersionTranslation>(runnableObject, getUserLanguages(session)).name ?? null}
                objectType={objectType}
                onSelect={handleRunSelect}
                runnableObject={runnableObject}
            />
            {/* Run button */}
            <Tooltip title="Run Routine" placement="top">
                {/* Button wrapped in div so it can be pressed when disabled */}
                <Box onClick={runStart}>
                    <SideActionsButton
                        aria-label="run-routine"
                        disabled={status === Status.Invalid}
                    >
                        <PlayIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </SideActionsButton>
                </Box>
            </Tooltip>
        </>
    );
}
