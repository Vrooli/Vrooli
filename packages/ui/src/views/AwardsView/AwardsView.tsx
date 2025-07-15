import Box from "@mui/material/Box";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Chip from "@mui/material/Chip";
import LinearProgress from "@mui/material/LinearProgress";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { AwardCategory, awardNames, endpointsAward, type Award, type AwardSearchInput, type AwardSearchResult, type TranslationKeyAward } from "@vrooli/shared";
import { type TFunction } from "i18next";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { CardGrid } from "../../components/lists/CardGrid/CardGrid.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SessionContext } from "../../contexts/session.js";
import { useFetch } from "../../hooks/useFetch.js";
import { IconCommon, IconRoutine } from "../../icons/Icons.js";
import { ScrollBox } from "../../styles.js";
import { type AwardDisplay, type ViewProps } from "../../types.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";

// Category array for sorting
const categoryList = Object.values(AwardCategory);
const PERCENTS = 100;

/**
 * Converts a queried award object into an AwardDisplay object.
 * @param award The queried award object.
 * @param t The translation function.
 * @returns The AwardDisplay object.
 */
export function awardToDisplay(award: Award, t: TFunction<"common", undefined, "common">): AwardDisplay {
    // Find earned tier
    const earnedTierDisplay = awardNames[award.category](award.progress, false);
    let earnedTier: AwardDisplay["earnedTier"] | undefined = undefined;
    if (earnedTierDisplay.name && earnedTierDisplay.body) {
        earnedTier = {
            title: t(`${earnedTierDisplay.name}`, { ns: "award", ...earnedTierDisplay.nameVariables }),
            description: t(`${earnedTierDisplay.body}`, { ns: "award", ...earnedTierDisplay.bodyVariables }),
            level: earnedTierDisplay.level,
        };
    }
    // Find next tier
    const nextTierDisplay = awardNames[award.category](award.progress, true);
    let nextTier: AwardDisplay["nextTier"] | undefined = undefined;
    // Only set nextTier if it's actually a higher level than the earned tier
    if (nextTierDisplay.name && nextTierDisplay.body &&
        (!earnedTier || nextTierDisplay.level > earnedTierDisplay.level)) {
        nextTier = {
            title: t(`${nextTierDisplay.name}`, { ns: "award", ...nextTierDisplay.nameVariables }),
            description: t(`${nextTierDisplay.body}`, { ns: "award", ...nextTierDisplay.bodyVariables }),
            level: nextTierDisplay.level,
        };
    }
    return {
        category: award.category,
        categoryDescription: t(`${award.category}Title`, { ns: "award", defaultValue: "" }),
        categoryTitle: t(`${award.category}Body`, { ns: "award", defaultValue: "" }),
        earnedTier,
        nextTier,
        progress: award.progress,
    };
}

function AwardCard({ award }: { award: AwardDisplay }) {
    const theme = useTheme();

    // Determine if this award is fully completed
    const isFullyCompleted = useMemo(() => {
        return Boolean(award.earnedTier) && !award.nextTier;
    }, [award.earnedTier, award.nextTier]);

    // Get category-specific icon
    const getCategoryIcon = (category: AwardCategory) => {
        switch (category) {
            case AwardCategory.RoutineCreate:
                return <IconRoutine name="Routine" size={24} />;
            case AwardCategory.RunRoutine:
                return <IconCommon name="Launch" size={24} />;
            case AwardCategory.RunProject:
                return <IconCommon name="Launch" size={24} />;
            case AwardCategory.ProjectCreate:
                return <IconCommon name="Project" size={24} />;
            case AwardCategory.Reputation:
                return <IconCommon name="Stats" size={24} />;
            case AwardCategory.Streak:
                return <IconCommon name="Today" size={24} />;
            case AwardCategory.CommentCreate:
                return <IconCommon name="Comment" size={24} />;
            case AwardCategory.ApiCreate:
                return <IconCommon name="Api" size={24} />;
            case AwardCategory.SmartContractCreate:
                return <IconCommon name="SmartContract" size={24} />;
            case AwardCategory.OrganizationCreate:
            case AwardCategory.OrganizationJoin:
                return <IconCommon name="Team" size={24} />;
            case AwardCategory.UserInvite:
                return <IconCommon name="User" size={24} />;
            case AwardCategory.ObjectBookmark:
                return <IconCommon name="BookmarkFilled" size={24} />;
            case AwardCategory.ObjectReact:
                return <IconCommon name="HeartFilled" size={24} />;
            case AwardCategory.PullRequestCreate:
            case AwardCategory.PullRequestComplete:
                return <IconCommon name="Share" size={24} />;
            case AwardCategory.IssueCreate:
                return <IconCommon name="Report" size={24} />;
            case AwardCategory.NoteCreate:
                return <IconCommon name="Note" size={24} />;
            case AwardCategory.StandardCreate:
                return <IconCommon name="Standard" size={24} />;
            case AwardCategory.ReportEnd:
            case AwardCategory.ReportContribute:
                return <IconCommon name="Warning" size={24} />;
            case AwardCategory.AccountNew:
                return <IconCommon name="CreateAccount" size={24} />;
            case AwardCategory.AccountAnniversary:
                return <IconCommon name="Award" size={24} />;
            default:
                return <IconCommon name="Award" size={24} />;
        }
    };

    // Determine current tier information
    const { currentTier, targetTier, tierLabel } = useMemo(() => {
        if (isFullyCompleted && award.earnedTier) {
            return {
                currentTier: award.earnedTier,
                targetTier: award.earnedTier,
                tierLabel: `Tier ${award.earnedTier.level}`,
            };
        } else if (award.nextTier) {
            const earnedLevel = award.earnedTier?.level || 0;
            return {
                currentTier: award.earnedTier || { title: award.categoryTitle, description: award.categoryDescription, level: 0 },
                targetTier: award.nextTier,
                tierLabel: earnedLevel > 0 ? `Tier ${earnedLevel} → ${award.nextTier.level}` : `Tier ${award.nextTier.level}`,
            };
        } else if (award.earnedTier) {
            return {
                currentTier: award.earnedTier,
                targetTier: award.earnedTier,
                tierLabel: `Tier ${award.earnedTier.level}`,
            };
        } else {
            return {
                currentTier: { title: award.categoryTitle, description: award.categoryDescription, level: 1 },
                targetTier: { title: award.categoryTitle, description: award.categoryDescription, level: 1 },
                tierLabel: "No progress",
            };
        }
    }, [award, isFullyCompleted]);

    // Calculate progress percentage
    const percentage = useMemo(() => {
        if (isFullyCompleted) return 100;
        if (!award.nextTier || targetTier.level === 0) return 0;
        return Math.min(100, Math.round((award.progress / targetTier.level) * PERCENTS));
    }, [award.progress, targetTier.level, isFullyCompleted, award.nextTier]);

    // Determine colors and status
    const { statusColor, statusText, progressColor } = useMemo(() => {
        if (isFullyCompleted) {
            return {
                statusColor: "success" as const,
                statusText: "Completed",
                progressColor: "success" as const,
            };
        } else if (percentage > 80) {
            return {
                statusColor: "warning" as const,
                statusText: "Almost there!",
                progressColor: "warning" as const,
            };
        } else if (percentage > 0) {
            return {
                statusColor: "primary" as const,
                statusText: "In Progress",
                progressColor: "primary" as const,
            };
        } else {
            return {
                statusColor: "default" as const,
                statusText: "Not Started",
                progressColor: "primary" as const,
            };
        }
    }, [isFullyCompleted, percentage]);

    return (
        <Card
            data-testid="award-card"
            data-category={award.category}
            data-completed={isFullyCompleted}
            data-progress={percentage}
            sx={{
                height: "100%",
                border: isFullyCompleted ? 2 : 1,
                borderColor: isFullyCompleted ? "success.main" : "divider",
                position: "relative",
                overflow: "visible",
            }}
        >
            {/* Completion indicator */}
            {isFullyCompleted && (
                <Box
                    sx={{
                        position: "absolute",
                        top: -8,
                        right: -8,
                        bgcolor: "success.main",
                        color: "success.contrastText",
                        borderRadius: "50%",
                        width: 24,
                        height: 24,
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        zIndex: 1,
                    }}
                >
                    <IconCommon name="Complete" size={16} />
                </Box>
            )}

            <CardContent sx={{ p: 3, height: "100%" }}>
                <Stack spacing={2} height="100%">
                    {/* Header with icon and status */}
                    <Stack direction="row" spacing={2} alignItems="flex-start">
                        <Box
                            sx={{
                                p: 1.5,
                                borderRadius: 2,
                                bgcolor: isFullyCompleted ? "success.main" : "primary.main",
                                color: isFullyCompleted ? "success.contrastText" : "primary.contrastText",
                                display: "flex",
                                alignItems: "center",
                                justifyContent: "center",
                                minWidth: 48,
                                minHeight: 48,
                            }}
                        >
                            {getCategoryIcon(award.category)}
                        </Box>
                        <Box flex={1}>
                            <Typography variant="h6" sx={{ fontWeight: 600, mb: 0.5 }}>
                                {currentTier.title}
                            </Typography>
                            <Stack direction="row" spacing={1} flexWrap="wrap">
                                <Chip
                                    data-testid="award-status-chip"
                                    label={statusText}
                                    color={statusColor}
                                    size="small"
                                />
                                <Chip
                                    data-testid="award-tier-chip"
                                    label={tierLabel}
                                    variant="outlined"
                                    size="small"
                                />
                            </Stack>
                        </Box>
                    </Stack>

                    {/* Description */}
                    <Typography
                        variant="body2"
                        color="text.secondary"
                        sx={{
                            flex: 1,
                            display: "-webkit-box",
                            WebkitLineClamp: 2,
                            WebkitBoxOrient: "vertical",
                            overflow: "hidden",
                        }}
                    >
                        {currentTier.description}
                    </Typography>

                    {/* Progress section */}
                    <Box>
                        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                            <Typography variant="body2" color="text.secondary">
                                Progress
                            </Typography>
                            <Typography variant="body2" fontWeight={500}>
                                {award.progress} / {targetTier.level}
                            </Typography>
                        </Stack>

                        <LinearProgress
                            data-testid="award-progress-bar"
                            variant="determinate"
                            value={percentage}
                            color={progressColor}
                            sx={{
                                height: 8,
                                borderRadius: 4,
                                bgcolor: theme.palette.grey?.[200] || "#f5f5f5",
                            }}
                        />

                        <Typography
                            variant="caption"
                            color="text.secondary"
                            sx={{ mt: 0.5, display: "block", textAlign: "center" }}
                        >
                            {percentage}% complete
                        </Typography>
                    </Box>
                </Stack>
            </CardContent>
        </Card>
    );
}

export function AwardsView(_props: ViewProps) {
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
        // Add to awards array, filter duplicates, and sort
        const myAwards = data.edges.map(e => e.node).map(a => awardToDisplay(a, t));
        setAwards(a => [...myAwards, ...a]
            .filter((award, index, self) => self.findIndex(a => a.category === award.category) === index));
    }, [data, lng, t]);

    // Sort awards: fully completed first, then by number of tiers completed (progress), then by category
    const sortedAwards = useMemo(() => {
        return [...awards].sort((a, b) => {
            // First: Fully completed awards (earnedTier exists and no nextTier)
            const aCompleted = Boolean(a.earnedTier) && !a.nextTier;
            const bCompleted = Boolean(b.earnedTier) && !b.nextTier;

            if (aCompleted && !bCompleted) return -1;
            if (!aCompleted && bCompleted) return 1;

            // Second: Sort by progress (number of tiers completed)
            const aProgress = a.earnedTier?.level || 0;
            const bProgress = b.earnedTier?.level || 0;

            if (aProgress !== bProgress) return bProgress - aProgress;

            // Third: Sort by raw progress number
            if (a.progress !== b.progress) return b.progress - a.progress;

            // Finally: Sort by category
            return categoryList.indexOf(a.category) - categoryList.indexOf(b.category);
        });
    }, [awards]);

    // Count completed and in-progress awards for summary
    const { completedCount, inProgressCount } = useMemo(() => {
        const completed = sortedAwards.filter(a => Boolean(a.earnedTier) && !a.nextTier).length;
        const inProgress = sortedAwards.filter(a => a.progress > 0 && (Boolean(a.nextTier) || !a.earnedTier)).length;
        return { completedCount: completed, inProgressCount: inProgress };
    }, [sortedAwards]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Navbar title={t("Award", { count: 2 })} />

                {/* Summary header */}
                <Box data-testid="awards-summary" sx={{ mb: 4, textAlign: "center" }}>
                    <Typography variant="h4" sx={{ mb: 2 }}>
                        🏆 Your Awards Progress
                    </Typography>
                    <Stack direction="row" spacing={2} justifyContent="center" flexWrap="wrap">
                        <Chip
                            data-testid="completed-count-chip"
                            label={`${completedCount} Completed`}
                            color="success"
                            variant="outlined"
                            icon={<IconCommon name="CheckCircle" size={18} />}
                        />
                        <Chip
                            data-testid="in-progress-count-chip"
                            label={`${inProgressCount} In Progress`}
                            color="primary"
                            variant="outlined"
                            icon={<IconCommon name="Timer" size={18} />}
                        />
                        <Chip
                            data-testid="total-count-chip"
                            label={`${sortedAwards.length} Total`}
                            color="default"
                            variant="outlined"
                            icon={<IconCommon name="Award" size={18} />}
                        />
                    </Stack>
                </Box>

                {/* Unified awards grid */}
                <CardGrid data-testid="awards-grid" minWidth={350}>
                    {sortedAwards.map((award) => (
                        <AwardCard
                            key={award.category}
                            award={award}
                        />
                    ))}
                </CardGrid>
            </ScrollBox>
        </PageContainer>
    );
}
