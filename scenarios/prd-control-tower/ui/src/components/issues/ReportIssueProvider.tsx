import { createContext, ReactNode, useContext, useMemo, useState } from 'react'
import type { IssueReportSeed } from '../../types'
import { IssueReportDialog } from './IssueReportDialog'
import { BulkIssueReportDialog } from './BulkIssueReportDialog'

interface ReportIssueContextValue {
  openIssueDialog: (seed: IssueReportSeed) => void
  openBulkIssueDialog: (seeds: IssueReportSeed[]) => void
  closeDialogs: () => void
}

const ReportIssueContext = createContext<ReportIssueContextValue | null>(null)

export function ReportIssueProvider({ children }: { children: ReactNode }) {
  const [activeSeed, setActiveSeed] = useState<IssueReportSeed | null>(null)
  const [bulkSeeds, setBulkSeeds] = useState<IssueReportSeed[] | null>(null)

  const value = useMemo<ReportIssueContextValue>(
    () => ({
      openIssueDialog: (seed) => {
        setBulkSeeds(null)
        setActiveSeed(seed)
      },
      openBulkIssueDialog: (seeds) => {
        if (!seeds || seeds.length === 0) {
          return
        }
        setActiveSeed(null)
        setBulkSeeds(seeds)
      },
      closeDialogs: () => {
        setActiveSeed(null)
        setBulkSeeds(null)
      },
    }),
    [],
  )

  return (
    <ReportIssueContext.Provider value={value}>
      {children}
      <IssueReportDialog open={Boolean(activeSeed)} seed={activeSeed} onOpenChange={(open) => !open && setActiveSeed(null)} />
      <BulkIssueReportDialog open={Boolean(bulkSeeds && bulkSeeds.length > 0)} seeds={bulkSeeds ?? []} onOpenChange={(open) => !open && setBulkSeeds(null)} />
    </ReportIssueContext.Provider>
  )
}

export function useReportIssueActions(): ReportIssueContextValue {
  const context = useContext(ReportIssueContext)
  if (!context) {
    throw new Error('useReportIssueActions must be used within ReportIssueProvider')
  }
  return context
}
