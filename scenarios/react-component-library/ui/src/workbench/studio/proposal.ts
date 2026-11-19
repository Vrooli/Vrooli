import type { DesignProposal } from "../designs";

/**
 * A proposal plus the two facts the server cannot know: which design it was
 * made for, and the exact document it saw. Applying is refused when either
 * changed, so a slow model response can never patch a different design.
 */
export interface StudioProposal extends DesignProposal {
  designId: string;
  signature: string;
}
