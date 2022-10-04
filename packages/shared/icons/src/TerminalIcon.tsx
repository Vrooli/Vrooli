import { SvgPath } from './base';
import { SvgProps } from './types';

export const TerminalIcon = (props: SvgProps) => (
    <SvgPath
        props={props}
        d="M20 4H4a2 2 0 0 0-2 2v12c0 1.1.9 2 2 2h16a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2zm0 14H4V8h16v10zm-2-1h-6v-2h6v2zM7.5 17l-1.4-1.4L8.7 13 6 10.4 7.5 9l4 4-4 4z"
    />
)