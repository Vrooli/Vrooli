import { endpointGetProjectVersion, endpointPostProjectVersionDirectory, LINKS, ProjectVersion, ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, uuid } from "@local/shared";
import { Box, IconButton, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SiteSearchBar } from "components/inputs/search";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ObjectListActions } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { useDeleter } from "hooks/useDeleter";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { AddIcon, DeleteIcon, EditIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ObjectAction } from "utils/actions/objectActions";
import { listToAutocomplete } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { deleteArrayIndex, updateArray } from "utils/shape/general";
import { shapeProjectVersionDirectory } from "utils/shape/models/projectVersionDirectory";
import { ProjectViewProps } from "../types";

export const ProjectView = ({
    display,
    isOpen,
    onClose,
}: ProjectViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const { object: existing, isLoading, setObject: setProjectVersion } = useObjectFromUrl<ProjectVersion>({
        ...endpointGetProjectVersion,
        objectType: "ProjectVersion",
    });

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { name, description } = useMemo(() => {
        const { name, description } = getTranslation(existing, [language]);
        return { name, description };
    }, [existing, language]);

    const [directories, setDirectories] = useState<ProjectVersionDirectory[]>([]);
    useEffect(() => {
        setDirectories(existing?.directories ?? []);
    }, [existing?.directories]);

    const onAction = useCallback((action: keyof ObjectListActions<ProjectVersionDirectory>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                const id = data[0] as string;
                setDirectories(curr => deleteArrayIndex(curr, curr.findIndex(item => item.id === id)));
                break;
            }
            case "Updated": {
                const updated = data[0] as ProjectVersionDirectory;
                setDirectories(curr => updateArray(curr, curr.findIndex(item => item.id === updated.id), updated));
                break;
            }
        }
    }, []);

    const actionData = useObjectActions({
        object: existing,
        objectType: "ProjectVersion",
        onAction,
        setLocation,
        setObject: setProjectVersion,
    });

    const [createDirectory, { loading: isCreating, errors: createErrors }] = useLazyFetch<ProjectVersionDirectoryCreateInput, ProjectVersionDirectory>(endpointPostProjectVersionDirectory);
    const addNewDirectory = useCallback(async (to: any) => {
        fetchLazyWrapper<ProjectVersionDirectoryCreateInput, ProjectVersionDirectory>({
            fetch: createDirectory,
            inputs: shapeProjectVersionDirectory.create({
                //TODO
                __typename: "ProjectVersionDirectory" as const,
                id: uuid(),
                isRoot: false,
                projectVersion: {
                    __typename: "ProjectVersion" as const,
                    id: existing?.id ?? ""
                },
            }),
            successCondition: (data) => data !== null,
            onSuccess: (data) => {
                setDirectories((prev) => [...prev, data]);
            },
        });
    }, [createDirectory, existing?.id]);

    // Search dialog to find objects to add to directory
    const hasSelectedObject = useRef(false);
    const [searchOpen, setSearchOpen] = useState(false);
    const openSearch = useCallback(() => { setSearchOpen(true); }, []);
    const closeSearch = useCallback((selectedObject?: any) => {
        setSearchOpen(false);
        hasSelectedObject.current = !!selectedObject;
        if (selectedObject) {
            addNewDirectory(selectedObject);
        }
    }, [addNewDirectory]);

    const [searchString, setSearchString] = useState("");
    const updateSearchString = useCallback((newString: string) => {
        setSearchString(newString);
    }, []);

    const onDirectorySelect = useCallback((data: any) => {
        console.log("onDirectorySelect", data);
    }, []);

    const autocompleteOptions = useMemo(() => listToAutocomplete(directories, getUserLanguages(session)), [directories, session]);

    const {
        handleDelete,
        DeleteDialogComponent,
    } = useDeleter({
        object: existing,
        objectType: "ProjectVersion",
        onActionComplete: () => {
            const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));
            if (hasPreviousPage) window.history.back();
            else setLocation(LINKS.Home, { replace: true });
        },
    });

    return (
        <>
            {DeleteDialogComponent}
            <FindObjectDialog
                find="List"
                isOpen={searchOpen}
                handleCancel={closeSearch}
                handleComplete={closeSearch}
            />
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(name, t("Project", { count: 1 }))}
                options={[{
                    Icon: DeleteIcon,
                    label: t("Delete"),
                    onClick: handleDelete,
                }]}
                below={<Box sx={{
                    width: "min(100%, 700px)",
                    margin: "auto",
                    marginTop: 2,
                    marginBottom: 2,
                    paddingLeft: 2,
                    paddingRight: 2,
                }}>
                    <SiteSearchBar
                        id={`${existing?.id ?? "directory-list"}-search-bar`}
                        placeholder={"SearchProject"}
                        loading={false}
                        value={searchString}
                        onChange={updateSearchString}
                        onInputChange={onDirectorySelect}
                        options={autocompleteOptions}
                        sxs={{ root: { width: "min(100%, 600px)", paddingLeft: 2, paddingRight: 2 } }}
                    />
                </Box>}
            />
            <ListContainer
                emptyText={t("NoResults", { ns: "error" })}
                isEmpty={directories.length === 0 && !isLoading}
            >
                <ObjectList
                    dummyItems={new Array(5).fill("ProjectVersionDirectory")}
                    items={directories}
                    keyPrefix="directory-list-item"
                    loading={isLoading}
                    onAction={onAction}
                />
            </ListContainer>
            <SideActionsButtons display={display} >
                <IconButton aria-label={t("UpdateList")} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} sx={{ background: palette.secondary.main }}>
                    <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
                <IconButton aria-label={t("AddDirectory")} onClick={openSearch} sx={{ background: palette.secondary.main }}>
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
            </SideActionsButtons>
        </>
    );
};
