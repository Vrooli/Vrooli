import { motion } from 'framer-motion'
import {
  Shield,
  Folder,
  FolderOpen,
  Edit,
  Lock
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { cn } from '@/lib/utils'
import type { Folder as FolderType } from '@/types'

/**
 * Get icon component for a folder based on its ID
 */
function getFolderIcon(folderId: string, isSelected: boolean) {
  switch (folderId) {
    case 'core':
      return <Shield className="h-4 w-4" />
    case 'drafts':
      return <Edit className="h-4 w-4" />
    default:
      return isSelected ? <FolderOpen className="h-4 w-4" /> : <Folder className="h-4 w-4" />
  }
}

/**
 * Get color for a folder based on its ID
 */
function getFolderColor(folderId: string): string {
  switch (folderId) {
    case 'core':
      return '#6366f1' // indigo
    case 'local':
      return '#10b981' // emerald
    case 'drafts':
      return '#f59e0b' // amber
    default:
      return '#6366f1'
  }
}

interface FolderTreeProps {
  folders: FolderType[]
  selectedFolder: FolderType | null
  onSelectFolder: (folder: FolderType | null) => void
  isLoading?: boolean
}

export function FolderTree({ folders, selectedFolder, onSelectFolder, isLoading }: FolderTreeProps) {
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
              className="text-primary"
            >
              <Folder className="h-5 w-5" />
            </motion.div>
            Folders
          </CardTitle>
        </div>
      </CardHeader>

      <CardContent className="space-y-2">
        {isLoading ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-14 rounded-lg bg-muted/50 animate-pulse" />
            ))}
          </div>
        ) : (
          <motion.div
            variants={containerVariants}
            initial="hidden"
            animate="visible"
            className="space-y-1"
          >
            {folders.map((folder) => {
              const isSelected = selectedFolder?.id === folder.id
              const color = getFolderColor(folder.id)

              return (
                <motion.div
                  key={folder.id}
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
                    onClick={() => onSelectFolder(folder)}
                    whileHover={{ scale: 1.02, y: -1 }}
                    whileTap={{ scale: 0.98 }}
                  >
                    {/* Background gradient */}
                    <motion.div
                      className="absolute inset-0 opacity-5"
                      style={{
                        background: `linear-gradient(135deg, ${color} 0%, transparent 100%)`
                      }}
                      initial={{ opacity: 0 }}
                      animate={{ opacity: isSelected ? 0.1 : 0.05 }}
                    />

                    <div className="relative p-3">
                      <div className="flex items-center gap-3">
                        {/* Folder icon */}
                        <motion.div
                          className="flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center text-white font-medium text-sm"
                          style={{ backgroundColor: color }}
                          whileHover={{ scale: 1.1, rotate: 5 }}
                        >
                          {getFolderIcon(folder.id, isSelected)}
                        </motion.div>

                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <h3 className="font-medium text-sm truncate">
                              {folder.name}
                            </h3>
                            {folder.readonly && (
                              <motion.div
                                initial={{ scale: 0 }}
                                animate={{ scale: 1 }}
                                title="Read-only"
                              >
                                <Lock className="h-3 w-3 text-muted-foreground" />
                              </motion.div>
                            )}
                          </div>

                          <motion.p
                            className="text-xs text-muted-foreground"
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            transition={{ delay: 0.1 }}
                          >
                            {folder.promptCount} {folder.promptCount === 1 ? 'prompt' : 'prompts'}
                          </motion.p>
                        </div>
                      </div>

                      {/* Description on hover */}
                      <motion.p
                        className="text-xs text-muted-foreground mt-2 line-clamp-1"
                        initial={{ opacity: 0, height: 0 }}
                        animate={{
                          opacity: isSelected ? 1 : 0,
                          height: isSelected ? 'auto' : 0
                        }}
                      >
                        {folder.description}
                      </motion.p>
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
        )}

        {folders.length === 0 && !isLoading && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="text-center py-8 text-muted-foreground"
          >
            <Folder className="h-12 w-12 mx-auto mb-3 opacity-50" />
            <p className="text-sm">No folders available</p>
          </motion.div>
        )}
      </CardContent>
    </Card>
  )
}
