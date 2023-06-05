import { useQuery } from "@apollo/client";
import { Award, AwardCategory, awardFindMany, AwardKey, AwardSearchInput, AwardSearchResult } from "@local/shared";
import { Stack } from "@mui/material";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { AwardCard } from "components/lists/AwardCard/AwardCard";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AwardDisplay, Wrap } from "types";
import { awardToDisplay } from "utils/display/awardsDisplay";
import { getUserLanguages } from "utils/display/translationTools";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { SessionContext } from "utils/SessionContext";
import { AwardsViewProps } from "views/types";

// Category array for sorting
const categoryList = Object.values(AwardCategory);

//TODO store tiers in @local/shared, so we can show tier progress and stuff
//TODO store title and description for category (i.e. no tier) in awards.json

export const AwardsView = ({
    display = "page",
    onClose,
}: AwardsViewProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    // All awards. Combined with you award progress fetched from the backend.
    const [awards, setAwards] = useState<AwardDisplay[]>(() => {
        // 0-progress awards may not be initialized in the backend, so we need to initialize them here
        const noProgressAwards = Object.values(AwardCategory).map((category) => ({
            category,
            title: t(`${category}UnearnedTitle` as AwardKey, { ns: "award" }),
            description: t(`${category}UnearnedBody` as AwardKey, { ns: "award" }),
            progress: 0,
        })) as Award[];
        return noProgressAwards.map(a => awardToDisplay(a, t));
    });
    const { data, refetch, loading, errors } = useQuery<Wrap<AwardSearchResult, "awards">, Wrap<AwardSearchInput, "input">>(awardFindMany, { variables: { input: {} }, errorPolicy: "all" });
    useDisplayServerError(errors);
    useEffect(() => {
        if (!data) return;
        // Add to awards array, and sort by award category
        const myAwards = data.awards.edges.map(e => e.node).map(a => awardToDisplay(a, t));
        setAwards(a => [...a, ...myAwards].sort((a, b) => categoryList.indexOf(a.category) - categoryList.indexOf(b.category)));
    }, [data, lng, t]);
    console.log(awards);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "Award",
                    titleVariables: { count: 2 },
                }}
            />
            <Stack direction="column" spacing={2} sx={{ margin: 2, padding: 1 }} >
                {/* Display earned awards as a list of tags. Press or hover to see description */}
                <ContentCollapse
                    isOpen={true}
                    title={t("Earned") + "🏆"}
                    sxs={{ titleContainer: { marginBottom: 2 } }}
                >
                    <CardGrid minWidth={200} disableMargin={true}>
                        {awards.filter(a => Boolean(a.earnedTier) && a.progress > 0).map((award) => (
                            <AwardCard
                                key={award.category}
                                award={award}
                                isEarned={true}
                            />
                        ))}
                    </CardGrid>
                </ContentCollapse>
                {/* Display progress of awards as cards */}
                <ContentCollapse
                    isOpen={true}
                    title={t("InProgress") + "🏃‍♂️"}
                    sxs={{ titleContainer: { marginBottom: 2 } }}
                >
                    <CardGrid minWidth={200} disableMargin={true}>
                        {awards.map((award) => (
                            <AwardCard
                                key={award.category}
                                award={award}
                                isEarned={false}
                            />
                        ))}
                    </CardGrid>
                </ContentCollapse>
            </Stack>
        </>
    );
};
