import React from 'react';
import { Sparkles, Video, LayoutGrid, BarChart3, ArrowDownToLine } from 'lucide-react';

export interface FeatureConfig {
  id: string;
  title: string;
  label: string;
  icon: React.ReactNode;
  gradient: string;
  accentColor: string;
}

export const FEATURE_CONFIGS: FeatureConfig[] = [
  {
    id: 'ai-powered',
    title: 'AI-Powered',
    label: 'AI generates your workflow',
    icon: <Sparkles size={16} />,
    gradient: 'from-purple-500/20 to-pink-500/20',
    accentColor: 'purple',
  },
  {
    id: 'record-mode',
    title: 'Record Mode',
    label: 'Record your browser actions',
    icon: <Video size={16} />,
    gradient: 'from-red-500/20 to-orange-500/20',
    accentColor: 'red',
  },
  {
    id: 'visual-builder',
    title: 'Visual Builder',
    label: 'Build with drag-and-drop',
    icon: <LayoutGrid size={16} />,
    gradient: 'from-blue-500/20 to-cyan-500/20',
    accentColor: 'blue',
  },
  {
    id: 'test-monitor',
    title: 'Test & Monitor',
    label: 'Watch executions live',
    icon: <BarChart3 size={16} />,
    gradient: 'from-green-500/20 to-emerald-500/20',
    accentColor: 'green',
  },
  {
    id: 'exports',
    title: 'Exports',
    label: 'Style & share replays',
    icon: <ArrowDownToLine size={16} />,
    gradient: 'from-amber-500/20 to-orange-500/20',
    accentColor: 'amber',
  },
];
