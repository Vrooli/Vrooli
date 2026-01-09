import type { LucideIcon } from 'lucide-react'
import { Brain, Cpu, Database, Network, Settings, Target, TrendingUp } from 'lucide-react'

const categoryIconMap: Record<string, LucideIcon> = {
  individual: Target,
  software: Cpu,
  manufacturing: Settings,
  healthcare: Database,
  finance: TrendingUp,
  education: Brain
}

export const getSectorIcon = (category: string): LucideIcon => categoryIconMap[category] || Network
