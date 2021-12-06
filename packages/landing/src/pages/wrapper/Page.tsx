import { useEffect } from 'react';

interface Props {
    title?: string;
    children: JSX.Element;
}

export const Page = ({
    title = '',
    children
}: Props) => {

    useEffect(() => {
        document.title = title;
    }, [title]);

    return children;
};