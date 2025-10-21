import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useQuery } from '@tanstack/react-query'
import { ThemeProvider } from './hooks/use-theme'
import { OptimizedMotionProvider } from './components/lazy/LazyMotion'
import { api } from './lib/api'
import type { Campaign, Prompt } from './types'

// Import components (we'll create these next)
import { Header } from './components/Header'
import { Sidebar } from './components/Sidebar'
import { CampaignTree } from './components/CampaignTree'
import { PromptList } from './components/PromptList'
import { PromptEditor } from './components/PromptEditor'
import { CampaignSkeleton, PromptListSkeleton, PromptEditorSkeleton } from './components/ui/skeleton'
import { FloatingAddButton } from './components/FloatingAddButton'
import { ErrorBoundary } from './components/ErrorBoundary'

function AppContent() {
  const [selectedCampaign, setSelectedCampaign] = useState<Campaign | null>(null)
  const [selectedPrompt, setSelectedPrompt] = useState<Prompt | null>(null)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [isMobile, setIsMobile] = useState(typeof window !== 'undefined' ? window.innerWidth < 1024 : false)
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false)

  // Fetch campaigns
  const { data: campaigns = [], isLoading: campaignsLoading } = useQuery({
    queryKey: ['campaigns'],
    queryFn: api.getCampaigns,
  })

  // Fetch prompts for selected campaign
  const { data: prompts = [], isLoading: promptsLoading } = useQuery({
    queryKey: ['prompts', selectedCampaign?.id],
    queryFn: () => selectedCampaign ? api.getCampaignPrompts(selectedCampaign.id) : Promise.resolve([]),
    enabled: !!selectedCampaign,
  })

  // Handle search
  const { data: searchResults = [] } = useQuery({
    queryKey: ['search', searchQuery],
    queryFn: () => api.searchPrompts(searchQuery),
    enabled: searchQuery.length > 2,
  })

  const displayPrompts = searchQuery.length > 2 ? searchResults : prompts

  useEffect(() => {
    const handleResize = () => {
      if (typeof window === 'undefined') {
        return
      }
      setIsMobile(window.innerWidth < 1024)
    }

    handleResize()
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  useEffect(() => {
    if (!isMobile) {
      setIsMobileSidebarOpen(false)
    }
  }, [isMobile])

  const renderCampaignSection = () => (
    <ErrorBoundary>
      {campaignsLoading ? (
        <div className="bg-card/50 backdrop-blur-sm border border-border/50 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-semibold flex items-center gap-2">
              📁 Campaigns
            </h2>
          </div>
          <CampaignSkeleton />
        </div>
      ) : (
        <CampaignTree
          campaigns={campaigns}
          selectedCampaign={selectedCampaign}
          onSelectCampaign={(campaign) => {
            setSelectedCampaign(campaign)
            setIsMobileSidebarOpen(false)
          }}
        />
      )}
    </ErrorBoundary>
  )

  return (
    <div className="flex h-screen bg-gradient-to-br from-slate-50 to-blue-50 dark:from-slate-950 dark:to-blue-950">
      {/* Sidebar */}
      {!isMobile && (
        <motion.aside
          initial={false}
          animate={{
            width: sidebarCollapsed ? 60 : 320,
          }}
          transition={{ duration: 0.3, ease: 'easeInOut' }}
          className="relative border-r border-border/50 bg-card/50 backdrop-blur-sm"
        >
          <Sidebar collapsed={sidebarCollapsed} onToggle={() => setSidebarCollapsed(!sidebarCollapsed)} />

          {!sidebarCollapsed && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.2 }}
              className="p-4 space-y-4"
            >
              {renderCampaignSection()}
            </motion.div>
          )}
        </motion.aside>
      )}

      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <Header 
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          selectedCampaign={selectedCampaign}
          showSidebarToggle={isMobile}
          onToggleSidebar={() => setIsMobileSidebarOpen(true)}
        />

        {/* Content Area */}
        <main className="flex-1 overflow-hidden">
          <div className="h-full grid grid-cols-1 lg:grid-cols-12 gap-6 p-6">
            {/* Prompt List */}
            <motion.div
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.1 }}
              className="lg:col-span-5 xl:col-span-4"
            >
              <ErrorBoundary>
                {!selectedCampaign && !searchQuery ? (
                  <div className="h-full bg-card/50 backdrop-blur-sm border border-border/50 rounded-lg p-6">
                    <div className="flex items-center justify-between mb-3">
                      <h2 className="text-lg font-semibold flex items-center gap-2">
                        📝 Prompts
                      </h2>
                    </div>
                    <PromptListSkeleton />
                  </div>
                ) : (
                  <PromptList
                    prompts={displayPrompts}
                    selectedPrompt={selectedPrompt}
                    onSelectPrompt={setSelectedPrompt}
                    campaignId={selectedCampaign?.id}
                    isLoading={promptsLoading}
                    searchQuery={searchQuery}
                  />
                )}
              </ErrorBoundary>
            </motion.div>

            {/* Prompt Editor */}
            <motion.div
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.2 }}
              className="lg:col-span-7 xl:col-span-8"
            >
              <ErrorBoundary>
                {!selectedPrompt ? (
                  <div className="h-full bg-card/50 backdrop-blur-sm border border-border/50 rounded-lg">
                    <PromptEditorSkeleton />
                  </div>
                ) : (
                  <PromptEditor
                    prompt={selectedPrompt}
                    onSave={(updatedPrompt) => {
                      setSelectedPrompt(updatedPrompt)
                      // Invalidate queries to refetch data
                    }}
                    onDelete={(_promptId) => {
                      setSelectedPrompt(null)
                      // Invalidate queries to refetch data
                    }}
                  />
                )}
              </ErrorBoundary>
            </motion.div>
          </div>
        </main>
      </div>
      
      {/* Floating Add Button */}
      <ErrorBoundary>
        <FloatingAddButton
          onPromptCreated={(newPrompt) => {
            // Auto-select the newly created prompt
            setSelectedPrompt(newPrompt)
            // If the prompt was created for a different campaign, switch to it
            const campaign = campaigns.find(c => c.id === newPrompt.campaign_id)
            if (campaign) {
              setSelectedCampaign(campaign)
            }
          }}
        />
      </ErrorBoundary>

      {/* Mobile Sidebar Overlay */}
      <AnimatePresence>
        {isMobile && isMobileSidebarOpen && (
          <motion.div
            className="fixed inset-0 z-50"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <motion.div
              className="absolute inset-0 bg-black/40"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              onClick={() => setIsMobileSidebarOpen(false)}
            />
            <motion.div
              initial={{ x: -40, opacity: 0 }}
              animate={{ x: 0, opacity: 1 }}
              exit={{ x: -40, opacity: 0 }}
              transition={{ type: 'spring', stiffness: 260, damping: 30 }}
              className="relative h-full w-[min(90vw,320px)] max-w-sm"
            >
              <div className="absolute left-0 top-0 h-full w-full">
                <div className="flex h-full flex-col border-r border-border/40 bg-card/95 backdrop-blur-xl shadow-2xl">
                  <Sidebar
                    collapsed={false}
                    onToggle={() => setIsMobileSidebarOpen(false)}
                    variant="floating"
                    onClose={() => setIsMobileSidebarOpen(false)}
                  />
                  <div className="flex-1 overflow-y-auto">
                    <div className="p-4 space-y-4">
                      {renderCampaignSection()}
                    </div>
                  </div>
                </div>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

export default function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider>
        <OptimizedMotionProvider>
          <ErrorBoundary>
            <AppContent />
          </ErrorBoundary>
        </OptimizedMotionProvider>
      </ThemeProvider>
    </ErrorBoundary>
  )
}
