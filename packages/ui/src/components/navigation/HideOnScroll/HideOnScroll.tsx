import { Slide, useScrollTrigger } from "@mui/material";
import { HideOnScrollProps } from "../types";

export const HideOnScroll = ({
    children,
}: HideOnScrollProps) => {
    const trigger = useScrollTrigger();
    return (
        <Slide appear={false} direction="down" in={!trigger}>
            {children}
        </Slide>
    );
}