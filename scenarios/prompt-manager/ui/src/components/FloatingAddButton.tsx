import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Plus, X, Send } from 'lucide-react'
import { useMutation, useQueryClient, useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { LoadingSpinner } from './ui/loading-spinner'
import { cn } from '@/lib/utils'
import type { Prompt } from '@/types'

interface FloatingAddButtonProps {
  onPromptCreated?: (prompt: Prompt) => void
}

export function FloatingAddButton({ onPromptCreated }: FloatingAddButtonProps) {
  const [isExpanded, setIsExpanded] = useState(false)
  const [formData, setFormData] = useState({
    campaignId: '',
    title: '',
    content: ''
  })
  
  const queryClient = useQueryClient()

  // Fetch campaigns for the selector
  const { data: campaigns = [] } = useQuery({
    queryKey: ['campaigns'],
    queryFn: api.getCampaigns,
  })

  const createPromptMutation = useMutation({
    mutationFn: (prompt: Omit<Prompt, 'id' | 'created_at' | 'updated_at' | 'usage_count'>) =>
      api.createPrompt(prompt),
    onSuccess: (newPrompt) => {
      queryClient.invalidateQueries({ queryKey: ['prompts'] })
      queryClient.invalidateQueries({ queryKey: ['campaigns'] })
      setIsExpanded(false)
      setFormData({ campaignId: '', title: '', content: '' })
      onPromptCreated?.(newPrompt)
    },
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!formData.campaignId || !formData.title.trim() || !formData.content.trim()) return
    
    await createPromptMutation.mutateAsync({
      campaign_id: formData.campaignId,
      title: formData.title,
      content: formData.content,
      variables: [],
      tags: [],
      is_favorite: false,
    })
  }

  const handleClose = () => {
    setIsExpanded(false)
    setFormData({ campaignId: '', title: '', content: '' })
  }

  return (
    <div className="fixed bottom-6 right-6 z-50">
      <AnimatePresence mode="wait">
        {!isExpanded ? (
          // Collapsed FAB
          <motion.div
            key="fab"
            initial={{ scale: 0, rotate: -180 }}
            animate={{ scale: 1, rotate: 0 }}
            exit={{ scale: 0, rotate: 180 }}
            transition={{ type: "spring", stiffness: 260, damping: 20 }}
          >
            <motion.div
              whileHover={{ scale: 1.1 }}
              whileTap={{ scale: 0.9 }}
            >
              <Button
                size="lg"
                className={cn(
                  "h-14 w-14 rounded-full shadow-lg",
                  "bg-gradient-to-r from-primary to-primary/80",
                  "hover:shadow-xl hover:shadow-primary/25",
                  "border-2 border-primary/20"
                )}
                onClick={() => setIsExpanded(true)}
              >
                <Plus className="h-6 w-6" />
              </Button>
            </motion.div>
          </motion.div>
        ) : (
          // Expanded form
          <motion.div
            key="form"
            initial={{ 
              scale: 0.8,
              opacity: 0,
              x: 20,
              transformOrigin: "bottom right"
            }}
            animate={{ 
              scale: 1,
              opacity: 1,
              x: 0
            }}
            exit={{ 
              scale: 0.8,
              opacity: 0,
              x: 20
            }}
            transition={{ type: "spring", stiffness: 300, damping: 30 }}
            className={cn(
              "bg-card/95 backdrop-blur-md border border-border/50 rounded-2xl shadow-2xl",
              "p-6 min-w-[380px] max-w-[420px]"
            )}
          >
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
              <motion.h3 
                className="text-lg font-semibold flex items-center gap-2"
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: 0.1 }}
              >
                ✨ New Prompt
              </motion.h3>
              <motion.div
                initial={{ opacity: 0, rotate: -90 }}
                animate={{ opacity: 1, rotate: 0 }}
                transition={{ delay: 0.2 }}
              >
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleClose}
                  className="h-8 w-8 p-0 rounded-full"
                >
                  <X className="h-4 w-4" />
                </Button>
              </motion.div>
            </div>

            {/* Form */}
            <form onSubmit={handleSubmit} className="space-y-4">
              {/* Campaign Selector */}
              <motion.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.2 }}
                className="space-y-2"
              >
                <Label htmlFor="campaign" className="text-sm font-medium">
                  Campaign
                </Label>
                <select
                  id="campaign"
                  value={formData.campaignId}
                  onChange={(e) => setFormData({ ...formData, campaignId: e.target.value })}
                  className={cn(
                    "w-full h-10 px-3 rounded-lg border border-border bg-background",
                    "text-sm focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
                    "appearance-none cursor-pointer"
                  )}
                  required
                >
                  <option value="">Select a campaign...</option>
                  {campaigns.map((campaign) => (
                    <option key={campaign.id} value={campaign.id}>
                      {campaign.name}
                    </option>
                  ))}
                </select>
              </motion.div>

              {/* Title Input */}
              <motion.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.3 }}
                className="space-y-2"
              >
                <Label htmlFor="title" className="text-sm font-medium">
                  Title
                </Label>
                <Input
                  id="title"
                  value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                  placeholder="Enter prompt title..."
                  className="h-10"
                  required
                />
              </motion.div>

              {/* Content Textarea */}
              <motion.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.4 }}
                className="space-y-2"
              >
                <Label htmlFor="content" className="text-sm font-medium">
                  Prompt Content
                </Label>
                <textarea
                  id="content"
                  value={formData.content}
                  onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                  placeholder="Write your prompt here..."
                  className={cn(
                    "w-full h-32 p-3 text-sm bg-background border border-border rounded-lg resize-none",
                    "focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                  )}
                  required
                />
              </motion.div>

              {/* Action Buttons */}
              <motion.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.5 }}
                className="flex gap-2 pt-2"
              >
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleClose}
                  className="flex-1 h-10"
                  disabled={createPromptMutation.isPending}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  className="flex-1 h-10 bg-gradient-to-r from-primary to-primary/80"
                  disabled={
                    !formData.campaignId || 
                    !formData.title.trim() || 
                    !formData.content.trim() ||
                    createPromptMutation.isPending
                  }
                >
                  {createPromptMutation.isPending ? (
                    <>
                      <LoadingSpinner size="sm" className="mr-2" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <Send className="h-4 w-4 mr-2" />
                      Create
                    </>
                  )}
                </Button>
              </motion.div>
            </form>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}