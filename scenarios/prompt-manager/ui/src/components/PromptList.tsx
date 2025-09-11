import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  Plus, 
  Clock, 
  TrendingUp, 
  FileText,
  Heart,
  MoreVertical
} from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { LoadingSpinner } from './ui/loading-spinner'
import { cn, formatRelativeTime, truncateText, estimateTokens } from '@/lib/utils'
import type { Prompt } from '@/types'

interface PromptListProps {
  prompts: Prompt[]
  selectedPrompt: Prompt | null
  onSelectPrompt: (prompt: Prompt) => void
  campaignId?: string
  isLoading?: boolean
  searchQuery?: string
}

export function PromptList({ 
  prompts, 
  selectedPrompt, 
  onSelectPrompt, 
  campaignId,
  isLoading,
  searchQuery 
}: PromptListProps) {
  const [isCreating, setIsCreating] = useState(false)
  const [newPrompt, setNewPrompt] = useState({ title: '', content: '' })
  const [filter, setFilter] = useState<'all' | 'favorites' | 'recent'>('all')
  
  const queryClient = useQueryClient()

  const createPromptMutation = useMutation({
    mutationFn: (prompt: Omit<Prompt, 'id' | 'created_at' | 'updated_at' | 'usage_count'>) =>
      api.createPrompt(prompt),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['prompts'] })
      setIsCreating(false)
      setNewPrompt({ title: '', content: '' })
    },
  })

  const handleCreatePrompt = async () => {
    if (!newPrompt.title.trim() || !newPrompt.content.trim() || !campaignId) return
    
    await createPromptMutation.mutateAsync({
      campaign_id: campaignId,
      title: newPrompt.title,
      content: newPrompt.content,
      variables: [],
      tags: [],
      is_favorite: false,
    })
  }

  // Filter prompts based on current filter
  const filteredPrompts = prompts.filter(prompt => {
    switch (filter) {
      case 'favorites':
        return prompt.is_favorite
      case 'recent':
        const weekAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)
        return new Date(prompt.updated_at) > weekAgo
      default:
        return true
    }
  })

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

  const filters = [
    { key: 'all', label: 'All Prompts', icon: FileText },
    { key: 'favorites', label: 'Favorites', icon: Heart },
    { key: 'recent', label: 'Recent', icon: Clock },
  ]

  return (
    <Card className="h-full flex flex-col bg-card/50 backdrop-blur-sm border-border/50">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <motion.div
              animate={{ scale: [1, 1.1, 1] }}
              transition={{ duration: 2, repeat: Infinity, repeatDelay: 4 }}
            >
              📝
            </motion.div>
            Prompts
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
          
          {campaignId && (
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
        
        {/* Filter tabs */}
        {!searchQuery && (
          <div className="flex gap-1 p-1 bg-muted/50 rounded-lg">
            {filters.map(({ key, label, icon: Icon }) => (
              <Button
                key={key}
                variant={filter === key ? "secondary" : "ghost"}
                size="sm"
                onClick={() => setFilter(key as any)}
                className={cn(
                  "flex-1 h-8 text-xs transition-all duration-200",
                  filter === key && "bg-background shadow-sm"
                )}
              >
                <Icon className="h-3 w-3 mr-1" />
                {label}
              </Button>
            ))}
          </div>
        )}
      </CardHeader>
      
      <CardContent className="flex-1 flex flex-col space-y-3 overflow-hidden">
        <AnimatePresence>
          {isCreating && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              className="space-y-3 p-4 rounded-lg bg-gradient-to-br from-muted/50 to-muted/30 border border-dashed border-border"
            >
              <Input
                value={newPrompt.title}
                onChange={(e) => setNewPrompt({ ...newPrompt, title: e.target.value })}
                placeholder="Prompt title..."
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
                  onClick={handleCreatePrompt}
                  disabled={!newPrompt.title.trim() || !newPrompt.content.trim() || createPromptMutation.isPending}
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
                    setNewPrompt({ title: '', content: '' })
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
          ) : filteredPrompts.length === 0 ? (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              className="text-center py-12 text-muted-foreground"
            >
              <FileText className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm">
                {searchQuery 
                  ? `No prompts found for "${searchQuery}"` 
                  : campaignId 
                    ? 'No prompts in this campaign yet'
                    : 'No prompts found'
                }
              </p>
              {campaignId && !searchQuery && (
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
              {filteredPrompts.map((prompt) => {
                const isSelected = selectedPrompt?.id === prompt.id
                const tokenCount = estimateTokens(prompt.content)
                
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
                              {prompt.title}
                            </h3>
                            
                            {prompt.is_favorite && (
                              <motion.div
                                initial={{ scale: 0 }}
                                animate={{ scale: 1 }}
                                whileHover={{ scale: 1.2 }}
                              >
                                <Heart className="h-3 w-3 text-red-500 fill-current" />
                              </motion.div>
                            )}
                          </div>
                          
                          <p className="text-xs text-muted-foreground line-clamp-2 mb-2">
                            {truncateText(prompt.content, 120)}
                          </p>
                          
                          <div className="flex items-center gap-3 text-xs text-muted-foreground">
                            <span className="flex items-center gap-1">
                              <TrendingUp className="h-3 w-3" />
                              {prompt.usage_count || 0} uses
                            </span>
                            <span className="flex items-center gap-1">
                              <FileText className="h-3 w-3" />
                              ~{tokenCount} tokens
                            </span>
                            <span>{formatRelativeTime(prompt.updated_at)}</span>
                          </div>
                        </div>
                        
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