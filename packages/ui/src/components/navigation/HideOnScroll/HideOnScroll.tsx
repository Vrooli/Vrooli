import { Slide, useScrollTrigger } from "@mui/material";

export interface HideOnScrollProps {
    children: JSX.Element;
    forceVisible?: boolean;
}

export const HideOnScroll = ({
    children,
    forceVisible = false,
}: HideOnScrollProps) => {
    const trigger = useScrollTrigger();
    const shouldDisplay = forceVisible ? true : !trigger;

    return (
        <Slide appear={false} direction="down" in={shouldDisplay}>
            {children}
        </Slide>
    );
};
