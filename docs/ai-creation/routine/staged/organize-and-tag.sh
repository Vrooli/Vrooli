#!/bin/bash
# Script to organize routines into category folders and add tags

# Map of routines to their categories and suggested tags
declare -A routine_categories=(
    # Meta-Intelligence
    ["memory-maintenance-system.json"]="meta-intelligence:meta-intelligence,memory-management,monitoring"
    ["memory-maintenance.json"]="meta-intelligence:meta-intelligence,memory-management,monitoring"
    ["swarm-memory-maintenance.json"]="meta-intelligence:meta-intelligence,memory-management,ai-agent"
    ["capability-gap-analysis.json"]="meta-intelligence:meta-intelligence,capability-analysis,analysis"
    ["anomaly-detector.json"]="meta-intelligence:meta-intelligence,anomaly-detection,monitoring"
    ["data-anomaly-detector.json"]="meta-intelligence:meta-intelligence,anomaly-detection,analysis"
    
    # Personal Development
    ["habit-tracking-coach.json"]="personal:personal-development,habit-formation,coaching"
    ["habit-tracker-coach.json"]="personal:personal-development,habit-formation,monitoring"
    ["habit-formation-coach.json"]="personal:personal-development,habit-formation,coaching"
    ["wellness-plan-creator.json"]="personal:personal-development,wellness,planning"
    ["mindfulness-practice-guide.json"]="personal:personal-development,mindfulness,coaching"
    ["personal-brand-builder.json"]="personal:personal-development,personal-branding,planning"
    ["personal-development-planner.json"]="personal:personal-development,planning,optimization"
    ["introspective-self-review.json"]="personal:personal-development,analysis,coaching"
    ["health-symptom-tracker.json"]="personal:personal-development,wellness,monitoring"
    
    # Professional
    ["career-change-navigator.json"]="professional:professional-growth,career-navigation,planning"
    ["interview-preparation-coach.json"]="professional:professional-growth,interview-prep,coaching"
    ["employee-onboarding-guide.json"]="professional:professional-growth,team-building,planning"
    ["team-role-optimizer.json"]="professional:professional-growth,team-building,optimization"
    ["role-and-team-optimizer.json"]="professional:professional-growth,team-building,optimization"
    ["public-speaking-coach.json"]="professional:professional-growth,leadership,coaching"
    
    # Innovation & Creativity
    ["creative-problem-solver.json"]="innovation:innovation-creativity,problem-solving,generation"
    ["brainstorming-buddy.json"]="innovation:innovation-creativity,brainstorming,generation"
    ["brainstorm-buddy.json"]="innovation:innovation-creativity,brainstorming,generation"
    ["innovation-workshop-facilitator.json"]="innovation:innovation-creativity,design-thinking,coaching"
    ["comprehensive-innovation-workshop.json"]="innovation:innovation-creativity,design-thinking,planning"
    ["problem-reframer.json"]="innovation:innovation-creativity,problem-solving,analysis"
    ["problem-solving-framework.json"]="innovation:innovation-creativity,problem-solving,planning"
    
    # Research & Intelligence
    ["multi-source-synthesizer.json"]="research:research-intelligence,multi-source-synthesis,synthesis"
    ["research-synthesis-workflow.json"]="research:research-intelligence,multi-source-synthesis,synthesis"
    ["comprehensive-research-synthesis.json"]="research:research-intelligence,multi-source-synthesis,analysis"
    ["fact-checking-agent.json"]="research:research-intelligence,fact-checking,analysis"
    ["competitive-analysis-framework.json"]="research:research-intelligence,competitive-analysis,analysis"
    ["market-trends-monitor.json"]="research:research-intelligence,market-research,monitoring"
    ["context-search-assistant.json"]="research:research-intelligence,analysis,automation"
    ["contextual-search-assistant.json"]="research:research-intelligence,analysis,automation"
    
    # Technical
    ["code-debugging-assistant.json"]="technical:technical-tools,debugging,automation"
    ["debugging-assistant.json"]="technical:technical-tools,debugging,automation"
    ["code-generator-tester.json"]="technical:technical-tools,code-generation,automation"
    ["api-doc-generator.json"]="technical:technical-tools,api-documentation,generation"
    ["api-documentation-generator.json"]="technical:technical-tools,api-documentation,generation"
    ["data-summarizer-visualizer.json"]="technical:technical-tools,data-processing,analysis"
    ["data-insight-extraction.json"]="technical:technical-tools,data-processing,analysis"
    
    # Financial
    ["budget-tracker-advisor.json"]="financial:financial-management,budget-optimization,monitoring"
    ["expense-tracker-analyzer.json"]="financial:financial-management,expense-tracking,analysis"
    ["investment-portfolio-analyzer.json"]="financial:financial-management,investment-analysis,analysis"
    ["financial-independence-planner.json"]="financial:financial-management,financial-planning,planning"
    ["comprehensive-financial-independence.json"]="financial:financial-management,financial-planning,planning"
    ["subscription-manager.json"]="financial:financial-management,expense-tracking,monitoring"
    ["subscription-tracking-manager.json"]="financial:financial-management,expense-tracking,monitoring"
    
    # Decision Making
    ["decision-support-framework.json"]="decision-making:decision-support,planning,analysis"
    ["advanced-decision-support.json"]="decision-making:decision-support,analysis,optimization"
    ["crisis-management-advisor.json"]="decision-making:decision-support,crisis-management,planning"
    ["comprehensive-crisis-management.json"]="decision-making:decision-support,crisis-management,planning"
    ["risk-assessment-agent.json"]="decision-making:decision-support,risk-assessment,analysis"
    ["what-if-scenario-planner.json"]="decision-making:decision-support,scenario-planning,planning"
    ["swot-analysis.json"]="decision-making:decision-support,swot-analysis,analysis"
    ["pros-cons-evaluator.json"]="decision-making:decision-support,analysis,planning"
    
    # Productivity
    ["task-prioritizer.json"]="productivity:productivity-optimization,task-prioritization,optimization"
    ["advanced-task-prioritizer.json"]="productivity:productivity-optimization,task-prioritization,optimization"
    ["time-block-scheduler.json"]="productivity:productivity-optimization,time-blocking,planning"
    ["calendar-time-blocker.json"]="productivity:productivity-optimization,time-blocking,automation"
    ["productivity-audit.json"]="productivity:productivity-optimization,analysis,optimization"
    ["time-management-optimizer.json"]="productivity:productivity-optimization,optimization,planning"
    ["project-planner-task-breakdown.json"]="productivity:productivity-optimization,project-management,planning"
    ["project-task-organizer.json"]="productivity:productivity-optimization,project-management,planning"
    ["daily-agenda-planner.json"]="productivity:productivity-optimization,planning,automation"
    ["weekly-review-assistant.json"]="productivity:productivity-optimization,planning,analysis"
    ["note-to-task-converter.json"]="productivity:productivity-optimization,task-prioritization,automation"
    ["priority-matrix.json"]="productivity:productivity-optimization,task-prioritization,analysis"
    ["dynamic-task-allocation.json"]="productivity:productivity-optimization,task-prioritization,ai-agent"
    ["goal-alignment-checkpoint.json"]="productivity:productivity-optimization,planning,analysis"
    
    # Learning
    ["learning-path-creator.json"]="learning:learning-education,adaptive-learning,planning"
    ["learning-plan-generator.json"]="learning:learning-education,adaptive-learning,generation"
    ["smart-study-planner.json"]="learning:learning-education,adaptive-learning,planning"
    ["study-session-planner.json"]="learning:learning-education,planning,optimization"
    ["language-learning-coach.json"]="learning:learning-education,language-learning,coaching"
    ["research-paper-assistant.json"]="learning:learning-education,knowledge-retention,generation"
    
    # Communication
    ["email-campaign-manager.json"]="communication:communication-content,email-management,automation"
    ["email-triage-assistant.json"]="communication:communication-content,email-management,automation"
    ["professional-report-writer.json"]="communication:communication-content,report-writing,generation"
    ["report-writer.json"]="communication:communication-content,report-writing,generation"
    ["content-outline-generator.json"]="communication:communication-content,content-creation,generation"
    ["social-media-content-planner.json"]="communication:communication-content,social-media,planning"
    ["social-media-post-composer.json"]="communication:communication-content,social-media,generation"
    ["meeting-minutes-summarizer.json"]="communication:communication-content,synthesis,automation"
    ["writing-coach-assistant.json"]="communication:communication-content,coaching,generation"
    ["essay-writing-coach.json"]="communication:communication-content,coaching,generation"
    ["creative-writing-mentor.json"]="communication:communication-content,coaching,generation"
    
    # Executive & Business
    ["strategic-planning-framework.json"]="executive:executive-leadership,strategic-vision,planning"
    ["customer-journey-mapper.json"]="business:business-development,analysis,planning"
    ["marketing-campaign-planner.json"]="business:business-development,planning,automation"
    
    # Lifestyle
    ["travel-planner.json"]="lifestyle:lifestyle-wellness,travel-planning,planning"
    ["travel-itinerary-builder.json"]="lifestyle:lifestyle-wellness,travel-planning,generation"
    ["home-maintenance-scheduler.json"]="lifestyle:lifestyle-wellness,home-management,planning"
    ["event-planning-assistant.json"]="lifestyle:lifestyle-wellness,planning,automation"
    ["relationship-communication-coach.json"]="lifestyle:lifestyle-wellness,coaching,optimization"
    ["relationship-communication-guide.json"]="lifestyle:lifestyle-wellness,coaching,planning"
    ["yes-man-avoidance.json"]="lifestyle:lifestyle-wellness,coaching,analysis"
    
    # Special/Other
    ["contract-review-assistant.json"]="business:business-development,analysis,automation"
    ["presentation-designer.json"]="communication:communication-content,generation,planning"
    ["conflict-resolution-mediator.json"]="professional:professional-growth,leadership,coaching"
)

# Function to add tags to a JSON file
add_tags_to_routine() {
    local file="$1"
    local tags="$2"
    
    # Convert comma-separated tags to JSON array with full tag objects
    local tag_array="[]"
    IFS=',' read -ra tag_names <<< "$tags"
    
    for tag_name in "${tag_names[@]}"; do
        # Look up the tag in our reference file
        tag_obj=$(jq -r --arg tag "$tag_name" '
            (.categories[].tags[], .["cross-cutting-tags"][]) | 
            select(.tag == $tag) | 
            @json
        ' tags-reference.json 2>/dev/null || echo "")
        
        if [ -n "$tag_obj" ]; then
            tag_array=$(echo "$tag_array" | jq --argjson obj "$tag_obj" '. += [$obj]')
        fi
    done
    
    # Update the file with the tags
    jq --argjson tags "$tag_array" '.tags = $tags' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
}

# Main processing
echo "Organizing routines into categories and adding tags..."

for routine_file in *.json; do
    # Skip reference files
    if [[ "$routine_file" == "tags-reference.json" || "$routine_file" == "category-tags.json" ]]; then
        continue
    fi
    
    # Get category and tags
    if [ -n "${routine_categories[$routine_file]}" ]; then
        IFS=':' read -r category tags <<< "${routine_categories[$routine_file]}"
        
        # Move to category folder
        if [ -d "$category" ]; then
            echo "Moving $routine_file to $category/ with tags: $tags"
            
            # Add tags first
            add_tags_to_routine "$routine_file" "$tags"
            
            # Then move the file
            mv "$routine_file" "$category/"
        else
            echo "Warning: Category folder $category not found for $routine_file"
        fi
    else
        echo "Warning: No category mapping for $routine_file"
    fi
done

# Move subroutines
if [ -d "subroutines" ]; then
    echo "Processing subroutines..."
    cd subroutines
    for subroutine in *.json; do
        # Add generic subroutine tag
        add_tags_to_routine "$subroutine" "automation"
    done
    cd ..
fi

echo "Organization complete!"