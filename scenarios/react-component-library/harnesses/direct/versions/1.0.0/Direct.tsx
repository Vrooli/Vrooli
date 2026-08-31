import type { ComponentType } from "react";
import { PreviewShowcase } from "../../../showcase/PreviewShowcase";

type DirectProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
};

export function Direct({ subject, args = {} }: DirectProps) {
  return <PreviewShowcase subject={subject} args={args} family="direct" />;
}
