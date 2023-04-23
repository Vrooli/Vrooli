import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useQuery } from "@apollo/client";
import { AwardCategory } from "@local/consts";
import { Stack } from "@mui/material";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { awardFindMany } from "../../api/generated/endpoints/award_findMany";
import { ContentCollapse } from "../../components/containers/ContentCollapse/ContentCollapse";
import { AwardCard } from "../../components/lists/AwardCard/AwardCard";
import { CardGrid } from "../../components/lists/CardGrid/CardGrid";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { awardToDisplay } from "../../utils/display/awardsDisplay";
import { getUserLanguages } from "../../utils/display/translationTools";
import { useDisplayApolloError } from "../../utils/hooks/useDisplayApolloError";
import { SessionContext } from "../../utils/SessionContext";
const categoryList = Object.values(AwardCategory);
export const AwardsView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);
    const [awards, setAwards] = useState(() => {
        const noProgressAwards = Object.values(AwardCategory).map((category) => ({
            category,
            title: t(`${category}UnearnedTitle`, { ns: "award" }),
            description: t(`${category}UnearnedBody`, { ns: "award" }),
            progress: 0,
        }));
        return noProgressAwards.map(a => awardToDisplay(a, t));
    });
    const { data, refetch, loading, error } = useQuery(awardFindMany, { variables: { input: {} }, errorPolicy: "all" });
    useDisplayApolloError(error);
    useEffect(() => {
        if (!data)
            return;
        const myAwards = data.awards.edges.map(e => e.node).map(a => awardToDisplay(a, t));
        setAwards(a => [...a, ...myAwards].sort((a, b) => categoryList.indexOf(a.category) - categoryList.indexOf(b.category)));
    }, [data, lng, t]);
    console.log(awards);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Award",
                    titleVariables: { count: 2 },
                } }), _jsxs(Stack, { direction: "column", spacing: 2, sx: { margin: 2, padding: 1 }, children: [_jsx(ContentCollapse, { isOpen: true, titleKey: "Earned", children: _jsx(CardGrid, { minWidth: 200, disableMargin: true, children: awards.filter(a => Boolean(a.earnedTier) && a.progress > 0).map((award) => (_jsx(AwardCard, { award: award, isEarned: true }, award.category))) }) }), _jsx(ContentCollapse, { isOpen: true, titleKey: "InProgress", children: _jsx(CardGrid, { minWidth: 200, disableMargin: true, children: awards.map((award) => (_jsx(AwardCard, { award: award, isEarned: false }, award.category))) }) })] })] }));
};
//# sourceMappingURL=AwardsView.js.map