import { SvgProps } from './types';

export const WholeWordIcon = (props: SvgProps) => (
    <svg
        id={props.id}
        style={props.style}
        width={props.width ?? '24px'}
        height={props.height ?? '24px'}
        viewBox="0 0 24 24"
        role="img"
        xmlSpace="preserve"
        xmlns="http://www.w3.org/2000/svg">
        <g
            aria-label="ab"
            transform="matrix(1.13298 0 0 1.02124 -.05 .1)"
            style={{
                stroke: props.fill ?? 'white',
                strokeWidth: ".93112",
                fontWeight: '300',
                fontSize: '16px',
                fontFamily: 'Lato',
                whiteSpace: 'pre',
            }}
        >
            <path d="M8.2 12.34q-1.13.04-1.95.19-.82.13-1.36.37-.52.24-.78.58-.26.33-.26.75 0 .4.13.7.14.29.35.48.23.18.52.27.28.09.6.09.46 0 .84-.1.4-.1.72-.28.34-.18.63-.43t.57-.54zM3.54 9.36q.62-.62 1.3-.94.7-.32 1.57-.32.64 0 1.12.2.48.2.79.59.32.37.48.92.16.53.16 1.2v5.18h-.3q-.24 0-.3-.22l-.11-1.03q-.33.32-.66.58-.32.26-.67.43-.35.18-.76.27-.4.1-.9.1-.4 0-.79-.12-.38-.12-.69-.37-.3-.25-.48-.63-.17-.4-.17-.94 0-.5.29-.93.28-.43.9-.75t1.58-.52q.96-.19 2.32-.22v-.83q0-1.1-.48-1.7-.48-.6-1.41-.6-.58 0-.99.16-.4.16-.68.35-.28.2-.45.36-.18.16-.3.16-.09 0-.14-.04-.06-.04-.1-.1zM12.26 14.46q.47.68 1.02.96.56.28 1.25.28.72 0 1.24-.26.54-.26.9-.73.35-.48.53-1.16.17-.68.17-1.53 0-1.64-.65-2.48-.66-.83-1.84-.83-.85 0-1.48.41-.63.4-1.14 1.14zm0-4.84q.52-.7 1.22-1.1.7-.42 1.61-.42.72 0 1.3.27.56.26.95.76.4.5.6 1.22.22.72.22 1.64 0 .97-.23 1.77-.24.8-.68 1.36-.44.56-1.08.87-.64.3-1.47.3-.85 0-1.45-.31-.59-.33-1.02-.97l-.06.98q-.02.2-.21.2h-.47V4.55h.77z" />
        </g>
        <path
            style={{
                fill: props.fill ?? 'white',
                stroke: props.fill ?? 'white',
                strokeWidth: "1.37953",
                strokeLinecap: "round",
            }}
            d="M.73 10.73v8.81M23.28 10.73v8.81M23.15 19.54H.75"
        />
    </svg>
)