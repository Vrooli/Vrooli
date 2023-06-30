import { SlideContainer, SlideContent } from "styles";
import { SlideProps } from "../types";

export const Slide = ({
    id,
    children,
    sx,
}: SlideProps) => {
    return (
        <SlideContainer id={id} sx={sx}>
            <SlideContent>
                {children}
            </SlideContent>
        </SlideContainer>
    );
};
