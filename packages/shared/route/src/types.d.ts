import { Path } from '.';

declare module '@shared/route';
export * from '.';

// Miscellaneous types
export type SetLocation = (to: Path, options?: { replace?: boolean }) => void;