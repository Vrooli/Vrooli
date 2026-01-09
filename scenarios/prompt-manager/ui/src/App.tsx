import { useEffect, useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useQuery } from '@tanstack/react-query'
import { FolderOpen, Star, Clock, TrendingUp, Sparkles } from 'lucide-react'
import { ThemeProvider } from './hooks/use-theme'
import { OptimizedMotionProvider } from './components/lazy/LazyMotion'
import { api } from './lib/api'
import type { Campaign, Prompt } from './types'

// Import components
import { Header } from './components/Header'
import { Sidebar, type ViewFilter } from './components/Sidebar'
import { CampaignTree } from './components/CampaignTree'
import { PromptList } from './components/PromptList'
import { PromptEditor } from './components/PromptEditor'
import { CampaignSkeleton, PromptEditorSkeleton } from './components/ui/skeleton'
import { FloatingAddButton } from './components/FloatingAddButton'
import { ErrorBoundary } from './components/ErrorBoundary'
import { Button } from './components/ui/button'

function AppContent() {
  const [selectedCampaign, setSelectedCampaign] = useState<Campaign | null>(null)
  const [selectedPrompt, setSelectedPrompt] = useState<Prompt | null>(null)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [viewFilter, setViewFilter] = useState<ViewFilter>('campaigns')
  const [isMobile, setIsMobile] = useState(typeof window !== 'undefined' ? window.innerWidth < 1024 : false)
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false)

  // Fetch campaigns
  const { data: campaigns = [], isLoading: campaignsLoading } = useQuery({
    queryKey: ['campaigns'],
    queryFn: api.getCampaigns,
  })

  // Fetch all prompts for filtering views
  const { data: allPrompts = [], isLoading: allPromptsLoading } = useQuery({
    queryKey: ['prompts', 'all'],
    queryFn: () => api.getPrompts(),
  })

  // Fetch prompts for selected campaign
  const { data: campaignPrompts = [], isLoading: campaignPromptsLoading } = useQuery({
    queryKey: ['prompts', 'campaign', selectedCampaign?.id],
    queryFn: () => selectedCampaign ? api.getCampaignPrompts(selectedCampaign.id) : Promise.resolve([]),
    enabled: !!selectedCampaign && viewFilter === 'campaigns',
  })

  // Handle search
  const { data: searchResults = [] } = useQuery({
    queryKey: ['search', searchQuery],
    queryFn: () => api.searchPrompts(searchQuery),
    enabled: searchQuery.length > 2,
  })

  // Compute filtered prompts based on view filter
  const filteredPrompts = useMemo(() => {
    if (searchQuery.length > 2) {
      return searchResults
    }

    switch (viewFilter) {
      case 'favorites':
        return allPrompts.filter(p => p.is_favorite)
      case 'recent': {
        const weekAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)
        return allPrompts
          .filter(p => new Date(p.updated_at) > weekAgo)
          .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
      }
      case 'popular':
        return [...allPrompts]
          .sort((a, b) => (b.usage_count || 0) - (a.usage_count || 0))
          .slice(0, 20)
      case 'campaigns':
      default:
        return selectedCampaign ? campaignPrompts : []
    }
  }, [viewFilter, allPrompts, campaignPrompts, searchResults, searchQuery, selectedCampaign])

  // Compute counts for sidebar badges
  const sidebarCounts = useMemo(() => {
    const weekAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)
    return {
      favorites: allPrompts.filter(p => p.is_favorite).length,
      recent: allPrompts.filter(p => new Date(p.updated_at) > weekAgo).length,
      popular: allPrompts.filter(p => (p.usage_count || 0) > 0).length,
    }
  }, [allPrompts])

  // Get filter display info
  const filterInfo = useMemo(() => {
    switch (viewFilter) {
      case 'favorites':
        return { icon: Star, label: 'Favorites', description: 'Your starred prompts' }
      case 'recent':
        return { icon: Clock, label: 'Recent', description: 'Updated in the last 7 days' }
      case 'popular':
        return { icon: TrendingUp, label: 'Popular', description: 'Most used prompts' }
      default:
        return null
    }
  }, [viewFilter])

  const isLoading = viewFilter === 'campaigns' ? campaignPromptsLoading : allPromptsLoading

  // Handle filter change - clear campaign selection when switching to non-campaign views
  const handleFilterChange = (filter: ViewFilter) => {
    setViewFilter(filter)
    if (filter !== 'campaigns') {
      setSelectedCampaign(null)
    }
    setIsMobileSidebarOpen(false)
  }

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
              <FolderOpen className="h-5 w-5 text-primary" />
              Campaigns
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
            setViewFilter('campaigns')
            setIsMobileSidebarOpen(false)
          }}
        />
      )}
    </ErrorBoundary>
  )

  // Render welcome state when no content is selected
  const renderWelcomeState = () => (
    <div className="h-full bg-card/50 backdrop-blur-sm border border-border/50 rounded-lg p-8 flex flex-col items-center justify-center text-center">
      <motion.div
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.3 }}
      >
        <div className="w-16 h-16 mx-auto mb-4 bg-gradient-to-br from-primary/20 to-primary/10 rounded-2xl flex items-center justify-center">
          <Sparkles className="h-8 w-8 text-primary" />
        </div>
        <h2 className="text-xl font-semibold mb-2">Welcome to Prompt Manager</h2>
        <p className="text-muted-foreground mb-6 max-w-md">
          Organize and manage your AI prompts with campaigns. Get started by selecting a campaign or exploring your prompts.
        </p>
        <div className="flex flex-wrap gap-3 justify-center">
          <Button
            variant="outline"
            onClick={() => setViewFilter('favorites')}
            className="gap-2"
          >
            <Star className="h-4 w-4" />
            View Favorites
          </Button>
          <Button
            variant="outline"
            onClick={() => setViewFilter('recent')}
            className="gap-2"
          >
            <Clock className="h-4 w-4" />
            Recent Prompts
          </Button>
        </div>
        {campaigns.length === 0 && !campaignsLoading && (
          <p className="text-sm text-muted-foreground mt-6">
            No campaigns yet. Click the + button to create your first campaign.
          </p>
        )}
      </motion.div>
    </div>
  )

  // Determine what to show in the prompt list area
  const showPromptList = searchQuery.length > 2 || viewFilter !== 'campaigns' || selectedCampaign

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
          <Sidebar
            collapsed={sidebarCollapsed}
            onToggle={() => setSidebarCollapsed(!sidebarCollapsed)}
            activeFilter={viewFilter}
            onFilterChange={handleFilterChange}
            counts={sidebarCounts}
          />

          {!sidebarCollapsed && viewFilter === 'campaigns' && (
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
                {!showPromptList ? (
                  renderWelcomeState()
                ) : (
                  <PromptList
                    prompts={filteredPrompts}
                    selectedPrompt={selectedPrompt}
                    onSelectPrompt={setSelectedPrompt}
                    campaignId={selectedCampaign?.id}
                    isLoading={isLoading}
                    searchQuery={searchQuery}
                    filterInfo={filterInfo}
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
                    }}
                    onDelete={() => {
                      setSelectedPrompt(null)
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
            setSelectedPrompt(newPrompt)
            const campaign = campaigns.find(c => c.id === newPrompt.campaign_id)
            if (campaign) {
              setSelectedCampaign(campaign)
              setViewFilter('campaigns')
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
                    activeFilter={viewFilter}
                    onFilterChange={handleFilterChange}
                    counts={sidebarCounts}
                  />
                  {viewFilter === 'campaigns' && (
                    <div className="flex-1 overflow-y-auto">
                      <div className="p-4 space-y-4">
                        {renderCampaignSection()}
                      </div>
                    </div>
                  )}
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
