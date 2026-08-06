/** @vrooliComponentSource primitives.icon */
interface BrandMarkProps {
  className?: string;
}

/** The React Component Library's composable-block mark. */
export function BrandMark({ className = "" }: BrandMarkProps) {
  return (
    <svg
      aria-hidden
      viewBox="0 0 32 32"
      className={className}
      focusable="false"
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect width="32" height="32" rx="9" fill="currentColor" />
      <path
        d="M8 8h7v7H8zM17 8h7v7h-7zM8 17h7v7H8z"
        fill="var(--color-primary-foreground, white)"
      />
      <path
        d="M17 18.5h7M20.5 15v7"
        stroke="var(--color-primary-foreground, white)"
        strokeWidth="2.5"
        strokeLinecap="round"
      />
    </svg>
  );
}
