#!/bin/bash

# Agent Import Script for Vrooli
# Imports AI-generated agents from staged directory into the database

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"

PROJECT_ROOT="${var_ROOT_DIR}"
AGENT_DIR="$PROJECT_ROOT/docs/ai-creation/agent"
STAGED_DIR="$AGENT_DIR/staged"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
    cat << EOF
Agent Import Script for Vrooli

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --dry-run           Show what would be imported without making changes
    --category TYPE     Import only specific category (coordinator|specialist|monitor|bridge)
    --file PATH         Import single agent file
    --limit N           Import only first N agents (for testing)
    --help              Show this help message

EXAMPLES:
    # Import all agents
    $0

    # Dry run to see what would be imported
    $0 --dry-run

    # Import only coordinator agents
    $0 --category coordinator

    # Import single agent
    $0 --file ./staged/coordinator/task-orchestration-coordinator.json

    # Import first 5 agents for testing
    $0 --limit 5
EOF
}

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $*" >&2
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
    exit 1
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

# Parse command line arguments
DRY_RUN=false
CATEGORY=""
SINGLE_FILE=""
LIMIT=0

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --category)
            CATEGORY="$2"
            shift 2
            ;;
        --file)
            SINGLE_FILE="$2"
            shift 2
            ;;
        --limit)
            LIMIT="$2"
            shift 2
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Check if development environment is running
check_environment() {
    log "Checking development environment..."
    
    if ! curl -s http://localhost:5329/health > /dev/null 2>&1; then
        error "Vrooli API is not running on localhost:5329. Start with: ./scripts/manage.sh develop --target docker --detached yes"
    fi
    
    success "API is running"
}

# Create TypeScript import script
create_import_script() {
    cat > "$PROJECT_ROOT/temp-agent-import.ts" << 'EOF'
import { PrismaClient } from '@prisma/client';
import fs from 'fs';
import path from 'path';

const prisma = new PrismaClient();

interface AgentImportOptions {
    dryRun: boolean;
    category?: string;
    singleFile?: string;
    limit?: number;
}

async function importAgents(options: AgentImportOptions) {
    const { dryRun, category, singleFile, limit } = options;
    
    let filesToImport: { path: string; category: string }[] = [];
    
    if (singleFile) {
        // Import single file
        const category = path.basename(path.dirname(singleFile));
        filesToImport.push({ path: singleFile, category });
    } else {
        // Import from staged directory
        const categories = category ? [category] : ['coordinator', 'specialist', 'monitor', 'bridge'];
        
        for (const cat of categories) {
            const dirPath = path.join('docs/ai-creation/agent/staged', cat);
            if (!fs.existsSync(dirPath)) continue;
            
            const files = fs.readdirSync(dirPath)
                .filter(f => f.endsWith('.json'))
                .map(f => ({ path: path.join(dirPath, f), category: cat }));
            
            filesToImport.push(...files);
        }
    }
    
    // Apply limit if specified
    if (limit && limit > 0) {
        filesToImport = filesToImport.slice(0, limit);
    }
    
    console.log(`Found ${filesToImport.length} agents to import`);
    
    let imported = 0;
    let updated = 0;
    let failed = 0;
    
    for (const { path: filePath, category } of filesToImport) {
        try {
            const agentData = JSON.parse(fs.readFileSync(filePath, 'utf8'));
            const agentName = agentData.identity?.name;
            
            if (!agentName) {
                console.error(`Skipping ${filePath}: No agent name found`);
                failed++;
                continue;
            }
            
            if (dryRun) {
                console.log(`[DRY RUN] Would import: ${agentName} (${category})`);
                imported++;
                continue;
            }
            
            // Check if bot already exists
            const existing = await prisma.user.findUnique({
                where: { handle: agentName }
            });
            
            const botSettings = {
                __version: "1.0",
                resources: agentData.resources || [],
                agentSpec: {
                    identity: agentData.identity,
                    goal: agentData.goal,
                    role: agentData.role,
                    subscriptions: agentData.subscriptions || [],
                    behaviors: agentData.behaviors || [],
                    prompt: agentData.prompt,
                    resources: agentData.resources || []
                }
            };
            
            if (existing) {
                console.log(`Updating agent: ${agentName}`);
                await prisma.user.update({
                    where: { id: existing.id },
                    data: { botSettings }
                });
                updated++;
            } else {
                console.log(`Creating agent: ${agentName}`);
                await prisma.user.create({
                    data: {
                        handle: agentName,
                        name: agentData.identity?.name || agentName,
                        isBot: true,
                        isPrivate: false,
                        botSettings
                    }
                });
                imported++;
            }
            
        } catch (error) {
            console.error(`Failed to import ${filePath}:`, error);
            failed++;
        }
    }
    
    console.log('\n=== Import Summary ===');
    console.log(`Imported: ${imported}`);
    console.log(`Updated: ${updated}`);
    console.log(`Failed: ${failed}`);
    console.log(`Total: ${filesToImport.length}`);
}

// Get options from environment variables
const options: AgentImportOptions = {
    dryRun: process.env.DRY_RUN === 'true',
    category: process.env.CATEGORY || undefined,
    singleFile: process.env.SINGLE_FILE || undefined,
    limit: process.env.LIMIT ? parseInt(process.env.LIMIT) : undefined
};

importAgents(options)
    .then(() => console.log('\nAgent import completed'))
    .catch(console.error)
    .finally(() => prisma.$disconnect());
EOF
}

# Run the import
run_import() {
    log "Creating import script..."
    create_import_script
    
    log "Running agent import..."
    
    # Set environment variables for the script
    export DRY_RUN="$DRY_RUN"
    export CATEGORY="$CATEGORY"
    export SINGLE_FILE="$SINGLE_FILE"
    export LIMIT="$LIMIT"
    
    # Run the import script
    cd "$PROJECT_ROOT/packages/server"
    npx tsx "$PROJECT_ROOT/temp-agent-import.ts"
    
    # Clean up
    rm -f "$PROJECT_ROOT/temp-agent-import.ts"
}

# Count agents by category
show_agent_stats() {
    log "Agent statistics:"
    echo ""
    
    local total=0
    for dir in "$STAGED_DIR"/*; do
        if [[ -d "$dir" ]]; then
            local category=$(basename "$dir")
            local count=$(find "$dir" -name "*.json" | wc -l)
            echo "  $category: $count agents"
            total=$((total + count))
        fi
    done
    
    echo ""
    echo "  Total: $total agents"
    echo ""
}

# Main execution
main() {
    log "Starting agent import process..."
    
    # Show stats
    show_agent_stats
    
    if [[ "$DRY_RUN" == "true" ]]; then
        warn "Running in DRY RUN mode - no changes will be made"
    fi
    
    # Check environment
    check_environment
    
    # Run import
    run_import
    
    if [[ "$DRY_RUN" == "false" ]]; then
        success "Agent import completed!"
        
        echo ""
        echo "Next steps:"
        echo "1. Verify agents in database:"
        echo "   docker compose exec db psql -U postgres -d vrooli -c \"SELECT handle, name FROM users WHERE \\\"isBot\\\" = true ORDER BY \\\"createdAt\\\" DESC LIMIT 10;\""
        echo "   (Note: If you get permission errors, try: sudo docker compose exec ... or newgrp docker)"
        echo ""
        echo "2. Test agent activation by creating a swarm"
        echo "3. Monitor agent behaviors in the logs"
    fi
}

main "$@"