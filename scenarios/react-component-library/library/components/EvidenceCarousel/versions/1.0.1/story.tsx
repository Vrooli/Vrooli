import { EvidenceCarousel, type EvidenceItem } from "./EvidenceCarousel";
export function EvidenceCarouselStory({ args }: StoryHarnessProps) {
  return <EvidenceCarousel items={args.items as EvidenceItem[] | undefined} />;
}
