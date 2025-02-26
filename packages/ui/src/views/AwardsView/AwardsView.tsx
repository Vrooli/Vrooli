import { Award, AwardCategory, AwardSearchInput, AwardSearchResult, TranslationKeyAward, endpointsAward } from "@local/shared";
import { Box, Typography } from "@mui/material";
import { CompletionBar } from "components/CompletionBar/CompletionBar";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse.js";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { SessionContext } from "contexts";
import { useFetch } from "hooks/useFetch";
import { AwardIcon } from "icons";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ScrollBox } from "styles";
import { AwardDisplay } from "types";
import { awardToDisplay } from "utils/display/awardsDisplay";
import { getUserLanguages } from "utils/display/translationTools.js";
import { AwardsViewProps } from "views/types";

// Category array for sorting
const categoryList = Object.values(AwardCategory);
const PERCENTS = 100;
const ALMOST_THERE_PERCENT = 90;

//TODO store tiers in @local/shared, so we can show tier progress and stuff
//TODO store title and description for category (i.e. no tier) in awards.json

const completionBarStyle = {
    root: { display: "block" },
    barBox: { maxWidth: "unset" },
} as const;
const almostThereStyle = { fontStyle: "italic" } as const;

function AwardCard({
    award,
    isEarned,
}: {
    award: AwardDisplay;
    isEarned: boolean;
}) {
    // Display highest earned tier or next tier,
    // depending on isEarned
    const { title, description, level } = useMemo(() => {
        // If not earned, display next tier
        if (!isEarned) {
            if (award.nextTier) return award.nextTier;
            // Default to earned tier if no next tier
            if (award.earnedTier) return award.earnedTier;
        }
        // If earned, display earned tier
        else if (award.earnedTier) return award.earnedTier;
        // If here, award invalid
        return { title: "", description: "", level: 0 };
    }, [award.earnedTier, award.nextTier, isEarned]);

    // Calculate percentage complete
    const percentage = useMemo(() => {
        if (award.progress === 0) return 0;
        if (level === 0) return -1;
        return Math.round((award.progress / level) * PERCENTS);
    }, [award.progress, level]);

    const isAlmostThere = useMemo(() => {
        return percentage > ALMOST_THERE_PERCENT && percentage < PERCENTS || level - award.progress === 1;
    }, [percentage, award.progress, level]);

    return (
        <TIDCard
            description={description}
            Icon={AwardIcon} //TODO Add custom icons/images for each category
            id={award.category}
            key={award.category}
            title={title}
            below={percentage >= 0 && <Box mt={2}>
                <CompletionBar
                    color={isAlmostThere ? "warning" : "secondary"}
                    showLabel={false}
                    value={percentage}
                    sxs={completionBarStyle}
                />
                <Typography
                    variant="body2"
                    component="p"
                    textAlign="center"
                    mt={1}
                    color="text.secondary"
                >
                    {award.progress} / {level} ({percentage}%)
                </Typography>
                {isAlmostThere &&
                    <Typography
                        variant="body2"
                        component="p"
                        textAlign="start"
                        mt={1}
                        style={almostThereStyle}
                    >
                        Almost there!
                    </Typography>
                }
            </Box>}
        />
    );
}

export function AwardsView({
    display,
    onClose,
}: AwardsViewProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    // All awards. Combined with you award progress fetched from the backend.
    const [awards, setAwards] = useState<AwardDisplay[]>(() => {
        // 0-progress awards may not be initialized in the backend, so we need to initialize them here
        const noProgressAwards = Object.values(AwardCategory).map((category) => ({
            category,
            title: t(`${category}UnearnedTitle` as TranslationKeyAward, { ns: "award" }),
            description: t(`${category}UnearnedBody` as TranslationKeyAward, { ns: "award" }),
            progress: 0,
        })) as Award[];
        return noProgressAwards.map(a => awardToDisplay(a, t));
    });
    const { data, refetch, loading } = useFetch<AwardSearchInput, AwardSearchResult>({
        ...endpointsAward.findMany,
    });
    useEffect(() => {
        if (!data) return;
        // Add to awards array, and sort by award category
        const myAwards = data.edges.map(e => e.node).map(a => awardToDisplay(a, t));
        setAwards(a => [...myAwards, ...a]
            .filter((award, index, self) => self.findIndex(a => a.category === award.category) === index)
            .sort((a, b) => categoryList.indexOf(a.category) - categoryList.indexOf(b.category)));
    }, [data, lng, t]);
    console.log(awards);

    return (
        <ScrollBox>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Award", { count: 2 })}
            />
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                gap: 4,
            }}>
                {/* Display earned awards as a list of tags. Press or hover to see description */}
                <ContentCollapse
                    isOpen={true}
                    title={t("Earned") + "🏆"}
                    sxs={{ titleContainer: { margin: 2, display: "flex", justifyContent: "center" } }}
                >
                    <CardGrid minWidth={300}>
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
                    sxs={{ titleContainer: { margin: 2, display: "flex", justifyContent: "center" } }}
                >
                    <CardGrid minWidth={300}>
                        {awards.map((award) => (
                            <AwardCard
                                key={award.category}
                                award={award}
                                isEarned={false}
                            />
                        ))}
                    </CardGrid>
                </ContentCollapse>
            </Box>
        </ScrollBox>
    );
}
