import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Plus,
  TrendingUp,
  FileText,
  Heart,
  MoreVertical,
  type LucideIcon
} from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { LoadingSpinner } from './ui/loading-spinner'
import { cn, formatRelativeTime, truncateText, estimateTokens } from '@/lib/utils'
import type { Prompt, CreatePromptRequest, FolderType } from '@/types'

interface FilterInfo {
  icon: LucideIcon
  label: string
  description: string
}

interface PromptListProps {
  prompts: Prompt[]
  selectedPrompt: Prompt | null
  onSelectPrompt: (prompt: Prompt) => void
  folder?: FolderType | null
  isReadonly?: boolean
  isLoading?: boolean
  searchQuery?: string
  filterInfo?: FilterInfo | null
  favorites: Set<string>
  onToggleFavorite: (id: string) => void
}

export function PromptList({
  prompts,
  selectedPrompt,
  onSelectPrompt,
  folder,
  isReadonly,
  isLoading,
  searchQuery,
  filterInfo,
  favorites,
  onToggleFavorite
}: PromptListProps) {
  const [isCreating, setIsCreating] = useState(false)
  const [newPrompt, setNewPrompt] = useState({ name: '', content: '' })

  const queryClient = useQueryClient()

  const createPromptMutation = useMutation({
    mutationFn: (prompt: CreatePromptRequest) => api.createPrompt(prompt),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['prompts'] })
      void queryClient.invalidateQueries({ queryKey: ['folders'] })
      setIsCreating(false)
      setNewPrompt({ name: '', content: '' })
    },
  })

  const handleCreatePrompt = async () => {
    if (!newPrompt.name.trim() || !newPrompt.content.trim() || !folder) return
    // Can only create in local or drafts
    if (folder !== 'local' && folder !== 'drafts') return

    await createPromptMutation.mutateAsync({
      name: newPrompt.name,
      description: '',
      content: newPrompt.content,
      folder: folder,
      modes: [],
      tags: [],
    })
  }

  const containerVariants = {
    hidden: { opacity: 0 },
    visible: {
      opacity: 1,
      transition: {
        delayChildren: 0.1,
        staggerChildren: 0.03
      }
    }
  }

  const itemVariants = {
    hidden: { opacity: 0, y: 20, scale: 0.95 },
    visible: {
      opacity: 1,
      y: 0,
      scale: 1,
      transition: { type: "spring", stiffness: 300, damping: 30 }
    }
  }

  // Determine the icon and label to show based on context
  const HeaderIcon = filterInfo?.icon ?? FileText
  const headerLabel = filterInfo?.label ?? 'Prompts'

  // Check if we can create prompts in this folder
  const canCreate = folder && !isReadonly && (folder === 'local' || folder === 'drafts')

  return (
    <Card className="h-full flex flex-col bg-card/50 backdrop-blur-sm border-border/50">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <CardTitle className="text-lg flex items-center gap-2">
              <motion.div
                animate={{ scale: [1, 1.1, 1] }}
                transition={{ duration: 2, repeat: Infinity, repeatDelay: 4 }}
                className="text-primary"
              >
                <HeaderIcon className="h-5 w-5" />
              </motion.div>
              {headerLabel}
              {searchQuery && (
                <motion.span
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  className="text-sm font-normal text-muted-foreground"
                >
                  • Search results
                </motion.span>
              )}
            </CardTitle>
            {filterInfo?.description && (
              <p className="text-xs text-muted-foreground mt-1 ml-7">
                {filterInfo.description}
              </p>
            )}
          </div>

          {canCreate && (
            <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button
                variant="glow"
                size="sm"
                onClick={() => setIsCreating(true)}
                className="h-8 w-8 p-0"
              >
                <Plus className="h-4 w-4" />
              </Button>
            </motion.div>
          )}
        </div>
      </CardHeader>

      <CardContent className="flex-1 flex flex-col space-y-3 overflow-hidden">
        <AnimatePresence>
          {isCreating && canCreate && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              className="space-y-3 p-4 rounded-lg bg-gradient-to-br from-muted/50 to-muted/30 border border-dashed border-border"
            >
              <Input
                value={newPrompt.name}
                onChange={(e) => setNewPrompt({ ...newPrompt, name: e.target.value })}
                placeholder="Prompt name..."
                className="h-9"
                autoFocus
              />
              <textarea
                value={newPrompt.content}
                onChange={(e) => setNewPrompt({ ...newPrompt, content: e.target.value })}
                placeholder="Write your prompt here..."
                className="w-full h-24 p-3 text-sm bg-background border border-border rounded-md resize-none focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              />
              <div className="flex gap-2">
                <Button
                  size="sm"
                  onClick={() => void handleCreatePrompt()}
                  disabled={!newPrompt.name.trim() || !newPrompt.content.trim() || createPromptMutation.isPending}
                  className="h-7 text-xs"
                >
                  {createPromptMutation.isPending ? (
                    <>
                      <LoadingSpinner size="sm" className="mr-1" />
                      Creating...
                    </>
                  ) : (
                    'Create'
                  )}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setIsCreating(false)
                    setNewPrompt({ name: '', content: '' })
                  }}
                  className="h-7 text-xs"
                >
                  Cancel
                </Button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Prompt list */}
        <div className="flex-1 overflow-y-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <LoadingSpinner size="lg" />
            </div>
          ) : prompts.length === 0 ? (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              className="text-center py-12 text-muted-foreground"
            >
              <FileText className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm">
                {searchQuery
                  ? `No prompts found for "${searchQuery}"`
                  : folder
                    ? 'No prompts in this folder yet'
                    : 'No prompts found'
                }
              </p>
              {canCreate && !searchQuery && (
                <p className="text-xs mt-1">Create your first prompt to get started</p>
              )}
            </motion.div>
          ) : (
            <motion.div
              variants={containerVariants}
              initial="hidden"
              animate="visible"
              className="space-y-2"
            >
              {prompts.map((prompt) => {
                const isSelected = selectedPrompt?.id === prompt.id
                const tokenCount = estimateTokens(prompt.content)
                const isFavorite = favorites.has(prompt.id)

                return (
                  <motion.div
                    key={prompt.id}
                    variants={itemVariants}
                    layout
                    onClick={() => onSelectPrompt(prompt)}
                    className={cn(
                      "group relative overflow-hidden rounded-lg border cursor-pointer transition-all duration-200",
                      isSelected
                        ? "border-primary bg-primary/5 shadow-lg shadow-primary/10"
                        : "border-border/50 bg-background/80 hover:bg-muted/30 hover:border-border"
                    )}
                    whileHover={{ y: -2, transition: { duration: 0.2 } }}
                    whileTap={{ scale: 0.98 }}
                  >
                    {/* Background pattern */}
                    <motion.div
                      className="absolute inset-0 opacity-[0.02]"
                      style={{
                        backgroundImage: `linear-gradient(45deg, transparent 25%, currentColor 25%, currentColor 50%, transparent 50%, transparent 75%, currentColor 75%)`,
                        backgroundSize: '8px 8px'
                      }}
                      initial={{ x: -8 }}
                      animate={{ x: 0 }}
                      transition={{ duration: 0.5, ease: "easeOut" }}
                    />

                    <div className="relative p-4">
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <h3 className="font-medium text-sm truncate">
                              {prompt.name}
                            </h3>

                            {isFavorite && (
                              <motion.div
                                initial={{ scale: 0 }}
                                animate={{ scale: 1 }}
                                whileHover={{ scale: 1.2 }}
                              >
                                <Heart className="h-3 w-3 text-red-500 fill-current" />
                              </motion.div>
                            )}

                            {prompt.draft && (
                              <span className="text-xs px-1.5 py-0.5 rounded bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300">
                                Draft
                              </span>
                            )}
                          </div>

                          <p className="text-xs text-muted-foreground line-clamp-2 mb-2">
                            {truncateText(prompt.description || prompt.content, 120)}
                          </p>

                          <div className="flex items-center gap-3 text-xs text-muted-foreground">
                            <span className="flex items-center gap-1">
                              <TrendingUp className="h-3 w-3" />
                              {prompt.usageCount} uses
                            </span>
                            <span className="flex items-center gap-1">
                              <FileText className="h-3 w-3" />
                              ~{tokenCount} tokens
                            </span>
                            <span>{formatRelativeTime(prompt.updatedAt)}</span>
                          </div>
                        </div>

                        <div className="flex items-center gap-1">
                          <motion.div
                            className="opacity-0 group-hover:opacity-100 transition-opacity"
                            onClick={(e) => {
                              e.stopPropagation()
                              onToggleFavorite(prompt.id)
                            }}
                          >
                            <Button
                              variant="ghost"
                              size="icon"
                              className={cn(
                                "h-6 w-6",
                                isFavorite && "text-red-500"
                              )}
                            >
                              <Heart className={cn("h-3 w-3", isFavorite && "fill-current")} />
                            </Button>
                          </motion.div>
                          <motion.div
                            className="opacity-0 group-hover:opacity-100 transition-opacity"
                            onClick={(e) => e.stopPropagation()}
                          >
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-6 w-6"
                            >
                              <MoreVertical className="h-3 w-3" />
                            </Button>
                          </motion.div>
                        </div>
                      </div>
                    </div>

                    {/* Selection indicator */}
                    {isSelected && (
                      <motion.div
                        className="absolute left-0 top-0 bottom-0 w-1 bg-gradient-to-b from-primary to-primary/50"
                        initial={{ scaleY: 0 }}
                        animate={{ scaleY: 1 }}
                        transition={{ type: "spring", stiffness: 400, damping: 30 }}
                      />
                    )}

                    {/* Hover effect */}
                    <motion.div
                      className="absolute inset-0 bg-gradient-to-r from-primary/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300"
                      initial={false}
                    />
                  </motion.div>
                )
              })}
            </motion.div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
