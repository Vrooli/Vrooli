import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  type CardProps,
} from "./Card";
import { CardGrid as CardGridSurface } from "@vrooli/react-component-library/CardGrid/1.0.0";

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

export function MetricCard() {
  return (
    <Card>
      <CardHeader><CardTitle>Adoption health</CardTitle></CardHeader>
      <CardContent>8 scenarios current</CardContent>
    </Card>
  );
}

export function EmptyTool() {
  return <Card><CardContent>No drift detected</CardContent></Card>;
}

export function CompactRecord() {
  return <Card><CardContent>Button / v1.2.0 / native</CardContent></Card>;
}

export function StandaloneFlat() {
  return <Card><CardContent>Standalone surface</CardContent></Card>;
}

export function CardGridRaised() {
  return <CardGridSurface><Card><CardContent>Grid surface</CardContent></Card></CardGridSurface>;
}
