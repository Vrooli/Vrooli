import { SvgProps } from './types';

export const UnlinkedNodesIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg">
        <path
            aria-label="Unlinked Nodes"
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.00157",
                strokeLinecap: "round",
                fontWeight: '300',
                fontSize: '11px',
                fontFamily: 'Lato',
            }}
            d="M10.37 4.85q.14-.14.32-.27.18-.14.4-.24.22-.1.47-.16.26-.07.56-.07.36 0 .7.12.33.1.58.32.25.2.4.5.14.3.14.68 0 .4-.13.7-.12.3-.31.52-.2.21-.42.38l-.42.32q-.2.14-.34.3-.13.14-.14.32l-.06.88h-.34l-.04-.91q0-.23.12-.4t.31-.32l.42-.32q.22-.16.41-.36.2-.2.33-.47.13-.26.13-.62 0-.29-.11-.51-.11-.23-.3-.39-.2-.16-.45-.24-.25-.08-.52-.08-.35 0-.61.1-.26.09-.44.2l-.27.21q-.1.1-.14.1-.08 0-.12-.07zm1.07 6.48q0-.1.04-.19t.1-.15q.07-.07.16-.1.09-.05.2-.05.1 0 .19.04t.15.1q.07.07.1.16.05.1.05.2t-.04.2q-.04.08-.1.15-.07.06-.16.1-.1.04-.2.04-.2 0-.35-.14-.14-.15-.14-.36z"
            fill="none"
        />
        <path
            style={{
                fillOpacity: 0,
                stroke: props.fill ?? 'white',
                strokeWidth: "1.75748",
                strokeLinecap: "round",
            }}
            d="M6.3 14.62h11.4m5.18 0a2.25 2.25 0 0 1-2.25 2.26 2.25 2.25 0 0 1-2.26-2.25 2.25 2.25 0 0 1 2.25-2.26 2.25 2.25 0 0 1 2.26 2.26Zm-17.22 0a2.25 2.25 0 0 1-2.25 2.26 2.25 2.25 0 0 1-2.25-2.25 2.25 2.25 0 0 1 2.25-2.26 2.25 2.25 0 0 1 2.25 2.26Z"
            fill="none"
        />
    </svg>
)