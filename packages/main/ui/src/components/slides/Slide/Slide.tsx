import { SlideContainer } from "../SlideContainer/SlideContainer";
import { SlideContent } from "../SlideContent/SlideContent";
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
