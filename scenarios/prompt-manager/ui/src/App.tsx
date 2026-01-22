import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { FolderOpen, Star, Clock, TrendingUp, Sparkles } from 'lucide-react'
import { ThemeProvider } from './hooks/use-theme'
import { useFavorites } from './hooks/use-favorites'
import { usePrompts } from './hooks/use-prompts'
import { OptimizedMotionProvider } from './components/lazy/LazyMotion'

// Import components
import { Header } from './components/Header'
import { Sidebar } from './components/Sidebar'
import { FolderTree } from './components/FolderTree'
import { PromptList } from './components/PromptList'
import { PromptEditor } from './components/PromptEditor'
import { FolderSkeleton, PromptEditorSkeleton } from './components/ui/skeleton'
import { FloatingAddButton } from './components/FloatingAddButton'
import { ErrorBoundary } from './components/ErrorBoundary'
import { Button } from './components/ui/button'

function AppContent() {
  // UI-only state (layout concerns)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [isMobile, setIsMobile] = useState(typeof window !== 'undefined' ? window.innerWidth < 1024 : false)
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false)

  // Favorites hook (local storage based)
  const { favorites, isFavorite, toggleFavorite } = useFavorites()

  // Prompts state management (extracted to dedicated hook)
  const {
    folders,
    filteredPrompts,
    sidebarCounts,
    selectedFolder,
    selectedPrompt,
    viewFilter,
    searchQuery,
    filterInfo,
    isLoading,
    foldersLoading,
    setSelectedFolder,
    setSelectedPrompt,
    setSearchQuery,
    handleFilterChange: baseHandleFilterChange,
    showPromptList,
  } = usePrompts({ favorites })

  // Wrap filter change to also close mobile sidebar
  const handleFilterChange = (filter: typeof viewFilter) => {
    baseHandleFilterChange(filter)
    setIsMobileSidebarOpen(false)
  }

  // Extended filter info with icons
  const filterInfoWithIcon = filterInfo ? {
    ...filterInfo,
    icon: filterInfo.label === 'Favorites' ? Star
      : filterInfo.label === 'Recent' ? Clock
      : TrendingUp
  } : null

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

  const renderFolderSection = () => (
    <ErrorBoundary>
      {foldersLoading ? (
        <div className="bg-card/50 backdrop-blur-sm border border-border/50 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <FolderOpen className="h-5 w-5 text-primary" />
              Folders
            </h2>
          </div>
          <FolderSkeleton />
        </div>
      ) : (
        <FolderTree
          folders={folders}
          selectedFolder={selectedFolder}
          onSelectFolder={(folder) => {
            setSelectedFolder(folder)
            handleFilterChange('folders')
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
          Organize and manage your AI prompts with folders. Get started by selecting a folder or exploring your prompts.
        </p>
        <div className="flex flex-wrap gap-3 justify-center">
          <Button
            variant="outline"
            onClick={() => handleFilterChange('favorites')}
            className="gap-2"
          >
            <Star className="h-4 w-4" />
            View Favorites
          </Button>
          <Button
            variant="outline"
            onClick={() => handleFilterChange('recent')}
            className="gap-2"
          >
            <Clock className="h-4 w-4" />
            Recent Prompts
          </Button>
        </div>
        {folders.length === 0 && !foldersLoading && (
          <p className="text-sm text-muted-foreground mt-6">
            No prompts yet. Click the + button to create your first prompt.
          </p>
        )}
      </motion.div>
    </div>
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
          <Sidebar
            collapsed={sidebarCollapsed}
            onToggle={() => setSidebarCollapsed(!sidebarCollapsed)}
            activeFilter={viewFilter}
            onFilterChange={handleFilterChange}
            counts={sidebarCounts}
          />

          {!sidebarCollapsed && viewFilter === 'folders' && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.2 }}
              className="p-4 space-y-4"
            >
              {renderFolderSection()}
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
          selectedFolder={selectedFolder}
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
                    folder={selectedFolder?.id}
                    isReadonly={selectedFolder?.readonly}
                    isLoading={isLoading}
                    searchQuery={searchQuery}
                    filterInfo={filterInfoWithIcon}
                    favorites={favorites}
                    onToggleFavorite={toggleFavorite}
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
                    isFavorite={isFavorite(selectedPrompt.id)}
                    onToggleFavorite={() => toggleFavorite(selectedPrompt.id)}
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
            // Find and select the folder the prompt was created in
            const folder = folders.find(f => f.id === newPrompt.folder)
            if (folder) {
              setSelectedFolder(folder)
              handleFilterChange('folders')
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
                  {viewFilter === 'folders' && (
                    <div className="flex-1 overflow-y-auto">
                      <div className="p-4 space-y-4">
                        {renderFolderSection()}
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
