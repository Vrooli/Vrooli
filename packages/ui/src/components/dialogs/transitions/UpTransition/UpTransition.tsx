import { Slide, SlideProps } from '@material-ui/core';
import { forwardRef } from 'react';

export const UpTransition = forwardRef<unknown, SlideProps>(function Transition(props, ref) {
    return <Slide direction="up" ref={ref} {...props} />;
});