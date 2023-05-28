import { ArrowLeftIcon, ArrowRightIcon, CompleteIcon, SetLocation, useLocation } from "@local/shared";
import { Box, IconButton, Palette, Stack, useTheme } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useCallback, useMemo, useState } from "react";
import { TutorialViewProps } from "../types";

type PageProps = {
    palette: Palette;
    setLocation: SetLocation;
}

// const Page0 = () => {
//     return (<>
//         <Header title="What is Vrooli?" />
//     </>)
// }

// const Page1 = () => {
//     return (<>
//         <Header title="How does it work?" />
//     </>)
// }

// const Page2 = () => {
//     return (<>
//         <Header title="What is a routine?" />
//     </>)
// }

// const Page3 = () => {
//     return (<>
//         <Header title="How is work structured?" />
//     </>)
// }

// const Page4 = () => {
//     return (<>
//         <Header title="LRD Process" />
//     </>)
// }

// const Page5 = () => {
//     return (<>
//         <Header title="That's the gist!" />
//     </>)
// }

/**
 * A 6-page tutorial for new users. Goes through the basics of Vrooli.
 * At the bottom of the page, there is a row of buttons to navigate between pages
 */
export const TutorialView = ({
    display = "page",
    onClose,
}: TutorialViewProps) => {
    const [, setLocation] = useLocation();
    const { palette } = useTheme();

    // 0-5
    const [page, setPage] = useState(0);
    const previousPage = useCallback(() => setPage(Math.max(0, page - 1)), [page, setPage]);
    const nextPage = useCallback(() => setPage(Math.min(5, page + 1)), [page, setPage]);
    const goToPage = useCallback((page: number) => setPage(page), [setPage]);

    const complete = useCallback(() => setLocation("/"), [setLocation]);

    const currentPage = useMemo(() => {
        // switch (page) {
        //     case 0: return <Page0 />;
        //     case 1: return <Page1 />;
        //     case 2: return <Page2 />;
        //     case 3: return <Page3 />;
        //     case 4: return <Page4 />;
        //     case 5: return <Page5 />;
        // }
        return null;
    }, [page]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "Tutorial",
                }}
            />
            {currentPage}
            {/* Buttons */}
            <Stack direction="row" spacing={2} sx={{ justifyContent: "center", alignItems: "center", textAlign: "center", marginTop: "1em" }}>
                {/* Previous arrow if available. If not, add empty space */}
                {page > 0 ? <IconButton onClick={previousPage}>
                    <ArrowLeftIcon fill={palette.background.textPrimary} />
                </IconButton> : <Box sx={{ width: "40px" }} />}
                {/* Dot for each page, with current page being larger, colored, and not selectable */}
                {[0, 1, 2, 3, 4, 5].map((p) => (
                    <Box key={p} sx={{
                        width: "1em",
                        height: "1em",
                        borderRadius: "100%",
                        background: p === page ? palette.primary.main : palette.background.textSecondary,
                        cursor: p === page ? "default" : "pointer",
                        transition: "0.3s ease-in-out",
                        "&:hover": {
                            transform: p === page ? "none" : "scale(1.2)",
                        },
                    }} onClick={() => goToPage(p)} />
                ))}
                {/* Next arrow if available. If not, complete arrow */}
                {page < 5 ? <IconButton onClick={nextPage}>
                    <ArrowRightIcon fill={palette.background.textPrimary} />
                </IconButton> : <IconButton onClick={complete}>
                    <CompleteIcon fill={palette.background.textPrimary} />
                </IconButton>}
            </Stack>
        </>
    );
};
