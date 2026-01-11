import { Globe, Layers, Server } from 'lucide-react';

export const SEGMENTS = [
  { id: 'apps' as const, label: 'Scenarios', icon: Layers },
  { id: 'resources' as const, label: 'Resources', icon: Server },
  { id: 'web' as const, label: 'Web Tabs', icon: Globe },
];

export type SegmentId = typeof SEGMENTS[number]['id'];

export const SEGMENT_QUERY_KEY = 'segment';
