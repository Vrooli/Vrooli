import { useEffect } from 'react';
import { PageProps } from './types';

export const Page = ({
    children,
    title = '',
}: PageProps) => {

    useEffect(() => {
        document.title = title;
    }, [title]);

    return children;
};