import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  Plus, 
  Folder, 
  FolderOpen, 
  MoreHorizontal,
  Heart
} from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { cn } from '@/lib/utils'
import type { Campaign } from '@/types'

interface CampaignTreeProps {
  campaigns: Campaign[]
  selectedCampaign: Campaign | null
  onSelectCampaign: (campaign: Campaign | null) => void
}

export function CampaignTree({ campaigns, selectedCampaign, onSelectCampaign }: CampaignTreeProps) {
  const [isCreating, setIsCreating] = useState(false)
  const [newCampaignName, setNewCampaignName] = useState('')
  const [expandedCampaigns, setExpandedCampaigns] = useState<Set<string>>(new Set())
  
  const queryClient = useQueryClient()

  const createCampaignMutation = useMutation({
    mutationFn: (campaign: Omit<Campaign, 'id' | 'created_at' | 'updated_at' | 'prompt_count'>) =>
      api.createCampaign(campaign),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['campaigns'] })
      setIsCreating(false)
      setNewCampaignName('')
    },
  })

  const handleCreateCampaign = async () => {
    if (!newCampaignName.trim()) return
    
    const colors = ['#6366f1', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#06b6d4']
    const icons = ['folder', 'star', 'heart', 'bookmark', 'tag']
    
    await createCampaignMutation.mutateAsync({
      name: newCampaignName,
      description: '',
      color: colors[Math.floor(Math.random() * colors.length)],
      icon: icons[Math.floor(Math.random() * icons.length)],
      is_favorite: false,
    })
  }

  const toggleExpanded = (campaignId: string) => {
    const newExpanded = new Set(expandedCampaigns)
    if (newExpanded.has(campaignId)) {
      newExpanded.delete(campaignId)
    } else {
      newExpanded.add(campaignId)
    }
    setExpandedCampaigns(newExpanded)
  }

  const containerVariants = {
    hidden: { opacity: 0 },
    visible: {
      opacity: 1,
      transition: {
        delayChildren: 0.1,
        staggerChildren: 0.05
      }
    }
  }

  const itemVariants = {
    hidden: { opacity: 0, y: 10 },
    visible: { 
      opacity: 1, 
      y: 0,
      transition: { type: "spring", stiffness: 300, damping: 30 }
    }
  }

  return (
    <Card className="h-full bg-card/50 backdrop-blur-sm border-border/50">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <motion.div
              animate={{ rotate: [0, 5, -5, 0] }}
              transition={{ duration: 2, repeat: Infinity, repeatDelay: 3 }}
            >
              📁
            </motion.div>
            Campaigns
          </CardTitle>
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
        </div>
      </CardHeader>
      
      <CardContent className="space-y-2">
        <AnimatePresence>
          {isCreating && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              className="space-y-2 p-3 rounded-lg bg-muted/50 border border-dashed border-border"
            >
              <Input
                value={newCampaignName}
                onChange={(e) => setNewCampaignName(e.target.value)}
                placeholder="Enter campaign name..."
                className="h-8 text-sm"
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleCreateCampaign()
                  if (e.key === 'Escape') {
                    setIsCreating(false)
                    setNewCampaignName('')
                  }
                }}
              />
              <div className="flex gap-2">
                <Button
                  size="sm"
                  onClick={handleCreateCampaign}
                  disabled={!newCampaignName.trim() || createCampaignMutation.isPending}
                  className="h-6 text-xs"
                >
                  Create
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setIsCreating(false)
                    setNewCampaignName('')
                  }}
                  className="h-6 text-xs"
                >
                  Cancel
                </Button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        <motion.div
          variants={containerVariants}
          initial="hidden"
          animate="visible"
          className="space-y-1 max-h-[60vh] overflow-y-auto"
        >
          {campaigns.map((campaign) => {
            const isSelected = selectedCampaign?.id === campaign.id
            const isExpanded = expandedCampaigns.has(campaign.id)
            
            return (
              <motion.div
                key={campaign.id}
                variants={itemVariants}
                layout
              >
                <motion.div
                  className={cn(
                    "group relative overflow-hidden rounded-lg border transition-all duration-200 cursor-pointer",
                    isSelected 
                      ? "border-primary bg-primary/10 shadow-lg shadow-primary/20" 
                      : "border-border/50 bg-background/50 hover:bg-muted/50 hover:border-border"
                  )}
                  onClick={() => onSelectCampaign(campaign)}
                  whileHover={{ scale: 1.02, y: -1 }}
                  whileTap={{ scale: 0.98 }}
                >
                  {/* Background gradient */}
                  <motion.div
                    className="absolute inset-0 opacity-5"
                    style={{ 
                      background: `linear-gradient(135deg, ${campaign.color} 0%, transparent 100%)` 
                    }}
                    initial={{ opacity: 0 }}
                    animate={{ opacity: isSelected ? 0.1 : 0.05 }}
                  />
                  
                  <div className="relative p-3">
                    <div className="flex items-center gap-3">
                      {/* Campaign icon */}
                      <motion.div
                        className="flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center text-white font-medium text-sm"
                        style={{ backgroundColor: campaign.color }}
                        whileHover={{ scale: 1.1, rotate: 5 }}
                      >
                        {isExpanded ? (
                          <FolderOpen className="h-4 w-4" />
                        ) : (
                          <Folder className="h-4 w-4" />
                        )}
                      </motion.div>
                      
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <h3 className="font-medium text-sm truncate">
                            {campaign.name}
                          </h3>
                          {campaign.is_favorite && (
                            <motion.div
                              initial={{ scale: 0 }}
                              animate={{ scale: 1 }}
                              whileHover={{ scale: 1.2 }}
                            >
                              <Heart className="h-3 w-3 text-red-500 fill-current" />
                            </motion.div>
                          )}
                        </div>
                        
                        <motion.p 
                          className="text-xs text-muted-foreground"
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                          transition={{ delay: 0.1 }}
                        >
                          {campaign.prompt_count || 0} prompts
                        </motion.p>
                      </div>
                      
                      {/* Actions */}
                      <motion.div
                        className="opacity-0 group-hover:opacity-100 transition-opacity"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-6 w-6"
                          onClick={() => toggleExpanded(campaign.id)}
                        >
                          <MoreHorizontal className="h-3 w-3" />
                        </Button>
                      </motion.div>
                    </div>
                  </div>
                  
                  {/* Selection indicator */}
                  {isSelected && (
                    <motion.div
                      className="absolute left-0 top-0 bottom-0 w-1 bg-primary"
                      initial={{ scaleY: 0 }}
                      animate={{ scaleY: 1 }}
                      transition={{ type: "spring", stiffness: 300, damping: 30 }}
                    />
                  )}
                </motion.div>
              </motion.div>
            )
          })}
        </motion.div>
        
        {campaigns.length === 0 && !isCreating && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="text-center py-8 text-muted-foreground"
          >
            <Folder className="h-12 w-12 mx-auto mb-3 opacity-50" />
            <p className="text-sm">No campaigns yet</p>
            <p className="text-xs">Create your first campaign to get started</p>
          </motion.div>
        )}
      </CardContent>
    </Card>
  )
}