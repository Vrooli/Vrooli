import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { CommentFor } from "@local/consts";
import { EditIcon } from "@local/icons";
import { Box, Stack, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { standardVersionFindOne } from "../../../api/generated/endpoints/standardVersion_findOne";
import { ColorIconButton } from "../../../components/buttons/ColorIconButton/ColorIconButton";
import { CommentContainer } from "../../../components/containers/CommentContainer/CommentContainer";
import { TextCollapse } from "../../../components/containers/TextCollapse/TextCollapse";
import { StandardInput } from "../../../components/inputs/standards/StandardInput/StandardInput";
import { ObjectActionsRow } from "../../../components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "../../../components/lists/resource";
import { smallHorizontalScrollbar } from "../../../components/lists/styles";
import { TagList } from "../../../components/lists/TagList/TagList";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { DateDisplay } from "../../../components/text/DateDisplay/DateDisplay";
import { ObjectTitle } from "../../../components/text/ObjectTitle/ObjectTitle";
import { VersionDisplay } from "../../../components/text/VersionDisplay/VersionDisplay";
import { standardInitialValues } from "../../../forms/StandardForm/StandardForm";
import { ObjectAction } from "../../../utils/actions/objectActions";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { useObjectFromUrl } from "../../../utils/hooks/useObjectFromUrl";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
const containerProps = (palette) => ({
    boxShadow: 1,
    background: palette.background.paper,
    borderRadius: 1,
    overflow: "overlay",
    marginTop: 4,
    marginBottom: 4,
    padding: 2,
});
export const StandardView = ({ display = "page", partialData, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { isLoading, object: existing, permissions, setObject: setStandardVersion } = useObjectFromUrl({
        query: standardVersionFindOne,
        partialData,
    });
    const availableLanguages = useMemo(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    const [language, setLanguage] = useState(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0)
            return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);
    const { description, name } = useMemo(() => {
        const { description, name } = getTranslation(existing ?? partialData, [language]);
        return { description, name };
    }, [existing, partialData, language]);
    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);
    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);
    const actionData = useObjectActions({
        object: existing,
        objectType: "Standard",
        openAddCommentDialog,
        setLocation,
        setObject: setStandardVersion,
    });
    const initialValues = useMemo(() => standardInitialValues(session, (existing ?? partialData)), [existing, partialData, session]);
    const resourceList = useMemo(() => initialValues.resourceList, [initialValues]);
    const tags = useMemo(() => initialValues.root?.tags, [initialValues]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Standard",
                } }), _jsxs(Box, { sx: {
                    marginLeft: "auto",
                    marginRight: "auto",
                    width: "min(100%, 700px)",
                    padding: 2,
                }, children: [_jsx(Stack, { direction: "row", spacing: 2, sx: {
                            position: "fixed",
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            zIndex: zIndex + 2,
                            bottom: 0,
                            right: 0,
                            marginBottom: {
                                xs: "calc(56px + 16px + env(safe-area-inset-bottom))",
                                md: "calc(16px + env(safe-area-inset-bottom))",
                            },
                            marginLeft: "calc(16px + env(safe-area-inset-left))",
                            marginRight: "calc(16px + env(safe-area-inset-right))",
                            height: "calc(64px + env(safe-area-inset-bottom))",
                        }, children: permissions.canUpdate ? (_jsx(ColorIconButton, { "aria-label": "confirm-title-change", background: palette.secondary.main, onClick: () => { actionData.onActionStart(ObjectAction.Edit); }, children: _jsx(EditIcon, { fill: palette.secondary.contrastText, width: '36px', height: '36px' }) })) : null }), _jsx(ObjectTitle, { language: language, languages: availableLanguages, loading: isLoading, title: name, setLanguage: setLanguage, translations: existing?.translations ?? [], zIndex: zIndex }), _jsx(RelationshipList, { isEditing: false, objectType: "Routine", zIndex: zIndex }), Array.isArray(resourceList?.resources) && resourceList.resources.length > 0 && _jsx(ResourceListHorizontal, { title: "Resources", list: resourceList, canUpdate: false, handleUpdate: () => { }, loading: isLoading, zIndex: zIndex }), _jsx(Box, { sx: containerProps(palette), children: _jsx(TextCollapse, { title: "Description", text: description, loading: isLoading, loadingLines: 2 }) }), _jsx(Stack, { direction: "column", spacing: 4, sx: containerProps(palette), children: _jsx(StandardInput, { disabled: true, fieldName: "preview", zIndex: zIndex }) }), Array.isArray(tags) && tags.length > 0 && _jsx(TagList, { maxCharacters: 30, parentId: existing?.id ?? "", tags: tags, sx: { ...smallHorizontalScrollbar(palette), marginTop: 4 } }), _jsxs(Stack, { direction: "row", spacing: 1, mt: 2, mb: 1, children: [_jsx(DateDisplay, { loading: isLoading, showIcon: true, timestamp: existing?.created_at }), _jsx(VersionDisplay, { currentVersion: existing, prefix: " - ", versions: existing?.root?.versions })] }), _jsx(ObjectActionsRow, { actionData: actionData, exclude: [ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp], object: existing, zIndex: zIndex }), _jsx(Box, { sx: containerProps(palette), children: _jsx(CommentContainer, { forceAddCommentOpen: isAddCommentOpen, language: language, objectId: existing?.id ?? "", objectType: CommentFor.StandardVersion, onAddCommentClose: closeAddCommentDialog, zIndex: zIndex }) })] })] }));
};
//# sourceMappingURL=StandardView.js.map