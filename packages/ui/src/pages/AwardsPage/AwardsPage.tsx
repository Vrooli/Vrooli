import { useQuery } from "@apollo/client";
import { Box } from "@mui/material";
import { Award, AwardCategory, AwardSearchInput, AwardSearchResult } from "@shared/consts";
import { awardFindMany } from "api/generated/endpoints/award";
import { AwardCard, AwardList, CardGrid, PageContainer, PageTitle } from "components";
import { AwardsPageProps } from "pages/types";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AwardDisplay, Wrap } from "types";
import { awardToDisplay, getUserLanguages, useDisplayApolloError } from "utils";

// Category array for sorting
const categoryList = Object.values(AwardCategory);

//TODO store tiers in @shared/consts, so we can show tier progress and stuff
//TODO store title and description for category (i.e. no tier) in awards.json

export function AwardsPage({
    session,
}: AwardsPageProps) {
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    // All awards. Combined with you award progress fetched from the backend.
    const [awards, setAwards] = useState<AwardDisplay[]>(() => {
        // 0-progress awards may not be initialized in the backend, so we need to initialize them here
        const noProgressAwards = Object.values(AwardCategory).map((category) => ({
            category: category,
            title: t(`award:${category}UnearnedTitle`, { lng }),
            description: t(`award:${category}UnearnedBody`, { lng }),
            progress: 0,
        })) as Award[];
        return noProgressAwards.map(a => awardToDisplay(a, t, lng));
    });
    const { data, refetch, loading, error } = useQuery<Wrap<AwardSearchResult, 'awards'>, Wrap<AwardSearchInput, 'input'>>(awardFindMany, { variables: { input: {} }, errorPolicy: 'all' });
    useDisplayApolloError(error);
    useEffect(() => {
        if (!data) return;
        // Add to awards array, and sort by award category
        const myAwards = data.awards.edges.map(e => e.node).map(a => awardToDisplay(a, t, lng));
        setAwards(a => [...a, ...myAwards].sort((a, b) => categoryList.indexOf(a.category) - categoryList.indexOf(b.category)));
    }, [data, lng, t]);
    console.log(awards);

    return (
        <PageContainer>
            <PageTitle titleKey='Award' titleVariables={{ count: 2 }} session={session} />
            {/* Display earned awards as a list of tags. Press or hover to see description */}
            <Box>
                <AwardList awards={awards.filter(a => Boolean(a.earnedTier))} />
            </Box>
            {/* Display progress of awards as cards */}
            <CardGrid minWidth={300}>
                {awards.map((award) => (
                    <AwardCard key={award.category} award={award} />
                ))}
            </CardGrid>
        </PageContainer>
    )
}