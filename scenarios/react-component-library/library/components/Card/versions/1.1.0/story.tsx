import { Card, type CardProps } from "./Card";
import { CardGrid as CardGridSurface } from "../../../CardGrid/versions/1.0.0/CardGrid";

export function Standalone({ args }: StoryHarnessProps) {
  return <Card {...(args as unknown as CardProps)} />;
}

export function CardGrid({ args }: StoryHarnessProps) {
  return (
    <CardGridSurface>
      <Card {...(args as unknown as CardProps)} />
    </CardGridSurface>
  );
}
