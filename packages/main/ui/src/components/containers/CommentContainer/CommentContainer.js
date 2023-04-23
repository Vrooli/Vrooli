import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { CreateIcon } from "@local/icons";
import { uuidValidate } from "@local/uuid";
import { Button, Stack, useTheme } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useFindMany } from "../../../utils/hooks/useFindMany";
import { useWindowSize } from "../../../utils/hooks/useWindowSize";
import { SearchButtons } from "../../buttons/SearchButtons/SearchButtons";
import { CommentUpsertInput } from "../../inputs/CommentUpsertInput/CommentUpsertInput";
import { CommentThread } from "../../lists/comment";
import { ContentCollapse } from "../ContentCollapse/ContentCollapse";
export function CommentContainer({ forceAddCommentOpen, isOpen, language, objectId, objectType, onAddCommentClose, zIndex, }) {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const { t } = useTranslation();
    const { advancedSearchParams, advancedSearchSchema, allData, setAdvancedSearchParams, setAllData, setSortBy, setTimeFrame, sortBy, sortByOptions, timeFrame, } = useFindMany({
        canSearch: uuidValidate(objectId),
        searchType: "Comment",
        resolve: (result) => result.threads,
        where: {
            [`${objectType.toLowerCase()}Id`]: objectId,
        },
    });
    const onCommentAdd = useCallback((comment) => {
        setAllData(curr => [{
                __typename: "CommentThread",
                comment: comment,
                childThreads: [],
                endCursor: null,
                totalInThread: 0,
            }, ...curr]);
    }, [setAllData]);
    const [isAddCommentOpen, setIsAddCommentOpen] = useState(isMobile);
    useEffect(() => { setIsAddCommentOpen(!isMobile || (forceAddCommentOpen === true)); }, [forceAddCommentOpen, isMobile]);
    const handleAddCommentOpen = useCallback(() => setIsAddCommentOpen(true), []);
    const handleAddCommentClose = useCallback(() => {
        setIsAddCommentOpen(false);
        if (onAddCommentClose)
            onAddCommentClose();
    }, [onAddCommentClose]);
    useEffect(() => {
        if (!forceAddCommentOpen || isMobile)
            return;
        const addCommentInput = document.getElementById("markdown-input-add-comment-root");
        if (addCommentInput) {
            addCommentInput.scrollIntoView({ behavior: "smooth" });
            addCommentInput.focus();
        }
    }, [forceAddCommentOpen, isMobile]);
    return (_jsxs(ContentCollapse, { isOpen: isOpen, title: "Comments", children: [isAddCommentOpen && _jsx(CommentUpsertInput, { comment: undefined, language: language, objectId: objectId, objectType: objectType, onCancel: handleAddCommentClose, onCompleted: onCommentAdd, parent: null, zIndex: zIndex }), allData.length > 0 ? _jsxs(_Fragment, { children: [_jsx(SearchButtons, { advancedSearchParams: advancedSearchParams, advancedSearchSchema: advancedSearchSchema, searchType: "Comment", setAdvancedSearchParams: setAdvancedSearchParams, setSortBy: setSortBy, setTimeFrame: setTimeFrame, sortBy: sortBy, sortByOptions: sortByOptions, timeFrame: timeFrame, zIndex: zIndex }), _jsx(Stack, { direction: "column", spacing: 2, children: allData.map((thread, index) => (_jsx(CommentThread, { canOpen: true, data: thread, language: language, zIndex: zIndex }, index))) })] }) : (!isAddCommentOpen && isMobile) ? _jsx(Button, { fullWidth: true, startIcon: _jsx(CreateIcon, {}), onClick: handleAddCommentOpen, sx: { marginTop: 2 }, children: t("AddComment") }) : null] }));
}
//# sourceMappingURL=CommentContainer.js.map