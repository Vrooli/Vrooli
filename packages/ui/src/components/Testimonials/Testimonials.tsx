import { Avatar, Box, Fade, SxProps, Typography, useTheme } from "@mui/material";
import Testimonial1 from "assets/img/Testimonial1.png";
import Testimonial2 from "assets/img/Testimonial2.png";
import { UserIcon } from "icons";
import { useEffect, useMemo, useRef, useState } from "react";
import { placeholderColor } from "utils/display/listTools";

const useOnScreen = () => {
    const ref = useRef(null);
    const [isIntersecting, setIntersecting] = useState(false);

    useEffect(() => {
        const currentRef = ref.current;

        const observer = new IntersectionObserver(
            ([entry]) => {
                setIntersecting(entry.isIntersecting);
            },
            {
                rootMargin: "0px 0px -100px 0px",
            },
        );

        if (currentRef) {
            observer.observe(currentRef);
        }

        return () => {
            if (currentRef) {
                observer.unobserve(currentRef);
            }
        };
    }, []);

    return [ref, isIntersecting] as const;
};

const Testimonial = ({
    author,
    text,
    src,
    alt,
    sx,
}: {
    author: string;
    text: string;
    src: string;
    alt: string;
    sx?: SxProps;
}) => {
    const { palette } = useTheme();
    const [ref, isVisible] = useOnScreen();
    const profileColors = useMemo(() => placeholderColor(), []);

    return (
        <Box ref={ref} sx={{
            position: "relative",
            display: "flex",
            flexDirection: "row",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 2,
            background: palette.background.paper + (palette.mode === "light" ? "44" : "c4"),
            color: "white",
            borderRadius: 2,
            padding: 1,
            boxShadow: 3,
            zIndex: 6,
            ...sx,
        }}>
            <Box sx={{ display: "flex", flexDirection: "row", alignItems: "center" }}>
                <Avatar
                    src={src}
                    alt={alt}
                    sx={{
                        backgroundColor: profileColors[0],
                        width: { xs: "50px", md: "75px" },
                        height: { xs: "50px", md: "75px" },
                        pointerEvents: "none",
                        marginRight: 2,
                    }}
                >
                    <UserIcon width="75%" height="75%" fill={profileColors[1]} />
                </Avatar>
                <Fade in={isVisible} timeout={1500}>
                    <Typography variant="h6">
                        "{text}"
                        <span style={{ fontStyle: "italic" }}> - {author}</span>
                    </Typography>
                </Fade>
            </Box>
        </Box>
    );
};

export const Testimonials = () => {
    return (
        <>
            <Testimonial
                text="I used to do things... manually! Can you imagine? Thanks to Vrooli, I've now forgotten how to use a pen."
                author="Jeremy P., former pen enthusiast"
                src={Testimonial1}
                alt="Jeremy P."
                sx={{ marginBottom: 4 }}
            />
            <Testimonial
                text="Set up Vrooli to manage my cat's Instagram. Now Mr. Whiskers has more followers than I do."
                author="Elena K., overshadowed by her cat"
                src={Testimonial2}
                alt="Elena K."
            />
            <Typography variant="body2" align="center" sx={{ marginBottom: 2, color: "grey" }}>
                *Testimonials are satirical in nature and are for illustrative purposes only.
            </Typography>
        </>
    );
};
