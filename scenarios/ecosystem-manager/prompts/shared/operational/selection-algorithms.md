# Selection Algorithms

## Purpose
Selection algorithms determine what work to tackle next. Smart selection maximizes value, minimizes conflicts, and ensures steady progress across the system.

## Core Selection Algorithm

### Multi-Factor Scoring
```python
def calculate_selection_score(item):
    """
    Comprehensive scoring algorithm for work selection
    """
    # Base factors
    impact = item.get('impact', 5)  # 1-10
    urgency = item.get('urgency', 'medium')
    success_probability = item.get('success_prob', 0.7)
    resource_cost = item.get('resource_cost', 'moderate')
    
    # Convert to numeric scores
    urgency_score = {
        'critical': 10, 'high': 7, 
        'medium': 5, 'low': 3
    }[urgency]
    
    cost_score = {
        'minimal': 10, 'moderate': 5, 'heavy': 2
    }[resource_cost]
    
    # Calculate base score
    base_score = (
        impact * 0.35 +           # 35% weight on impact
        urgency_score * 0.25 +     # 25% weight on urgency
        success_probability * 10 * 0.20 +  # 20% weight on success likelihood
        cost_score * 0.20          # 20% weight on inverse cost
    )
    
    # Apply modifiers
    score = apply_modifiers(base_score, item)
    
    return score

def apply_modifiers(base_score, item):
    """
    Apply contextual modifiers to base score
    """
    score = base_score
    
    # Cross-scenario impact boost
    if item.get('cross_scenario_count', 0) > 3:
        score *= 1.5  # 50% boost for high cross-impact
    
    # Blocker boost
    if item.get('unblocks_count', 0) > 0:
        score *= (1 + 0.2 * item['unblocks_count'])  # 20% per unblocked item
    
    # Failed attempt penalty
    if item.get('attempt_count', 0) > 0:
        score *= (0.8 ** item['attempt_count'])  # 20% reduction per failure
    
    # Freshness boost (prefer newer items slightly)
    age_days = item.get('age_days', 0)
    if age_days < 1:
        score *= 1.1  # 10% boost for fresh items
    elif age_days > 7:
        score *= 0.9  # 10% penalty for stale items
    
    return score
```

## Selection Strategies

### Strategy 1: Value Maximization
```python
def select_max_value(available_items, time_budget):
    """
    Select items that maximize total value within time budget
    """
    # Score all items
    for item in available_items:
        item['score'] = calculate_selection_score(item)
        item['value_per_hour'] = item['score'] / item.get('estimated_hours', 1)
    
    # Sort by value per hour
    sorted_items = sorted(available_items, 
                         key=lambda x: x['value_per_hour'], 
                         reverse=True)
    
    # Select items that fit in budget
    selected = []
    remaining_time = time_budget
    
    for item in sorted_items:
        if item['estimated_hours'] <= remaining_time:
            selected.append(item)
            remaining_time -= item['estimated_hours']
            
    return selected
```

### Strategy 2: Quick Wins First
```python
def select_quick_wins(available_items, max_items=3):
    """
    Select high-value items that can be completed quickly
    """
    # Filter for quick items (< 2 hours)
    quick_items = [i for i in available_items 
                   if i.get('estimated_hours', 999) < 2]
    
    # Score and sort
    for item in quick_items:
        item['score'] = calculate_selection_score(item)
    
    sorted_items = sorted(quick_items, 
                         key=lambda x: x['score'], 
                         reverse=True)
    
    return sorted_items[:max_items]
```

### Strategy 3: Dependency Resolution
```python
def select_unblockers(available_items):
    """
    Prioritize items that unblock other work
    """
    # Find all blockers
    blockers = [i for i in available_items 
                if i.get('unblocks_count', 0) > 0]
    
    # Sort by number of items unblocked
    sorted_blockers = sorted(blockers,
                            key=lambda x: x['unblocks_count'],
                            reverse=True)
    
    if sorted_blockers:
        return sorted_blockers[0]  # Take biggest blocker
    
    # Fall back to normal selection
    return select_max_value(available_items, 4)[0]
```

## Collision Avoidance

### Cooldown System
```bash
# Calculate cooldown based on concurrent workers
calculate_cooldown() {
    local max_workers=${MAX_CONCURRENT_WORKERS:-3}
    local iteration_hours=${ITERATION_HOURS:-4}
    
    # Cooldown = time for all workers to complete one iteration
    local cooldown_hours=$((max_workers * iteration_hours))
    
    # Add random jitter to prevent thundering herd
    local jitter=$((RANDOM % 60))  # 0-60 minutes
    
    echo "$((cooldown_hours * 60 + jitter))"  # Return minutes
}

# Check if item is available (not in cooldown)
is_available() {
    local item=$1
    local cooldown_until=$(yq '.metadata.cooldown_until' $item)
    local now=$(date +%s)
    local cooldown_timestamp=$(date -d "$cooldown_until" +%s)
    
    if [ $now -gt $cooldown_timestamp ]; then
        return 0  # Available
    else
        return 1  # Still cooling down
    fi
}
```

### Worker Coordination
```python
def coordinate_selection(worker_id, available_items):
    """
    Coordinate selection across multiple workers
    """
    # Get other workers' current items
    in_progress = get_in_progress_items()
    other_workers = [i for i in in_progress 
                    if i['assigned_to'] != worker_id]
    
    # Filter out related items
    filtered = []
    for item in available_items:
        # Skip if another worker is working on same target
        if not any(w['target'] == item['target'] 
                  for w in other_workers):
            filtered.append(item)
    
    # Apply staggered selection based on worker ID
    # This prevents workers from always picking same items
    worker_offset = hash(worker_id) % 3
    if worker_offset == 0:
        return select_max_value(filtered, 4)
    elif worker_offset == 1:
        return select_quick_wins(filtered, 1)
    else:
        return select_unblockers(filtered)
```

## Selection Rubrics

### Scenario Selection Rubric
```markdown
## Scenario Selection Priority

1. **Critical Issues** (Select immediately)
   - Security vulnerabilities
   - Data corruption risks
   - Revenue-blocking bugs
   - System stability issues

2. **Cross-Scenario Impact** (High priority)
   - Shared workflow improvements
   - Common component enhancements
   - Pattern standardization
   - Resource optimization

3. **User-Facing Value** (Medium-high priority)
   - Features users actively request
   - UX improvements for common workflows
   - Performance improvements >20%
   - Documentation for complex features

4. **Technical Debt** (Medium priority)
   - Code quality improvements
   - Test coverage increases
   - Refactoring for maintainability
   - Dependency updates

5. **Nice-to-Have** (Low priority)
   - Cosmetic improvements
   - Minor optimizations <10%
   - Experimental features
   - Future-proofing
```

### Resource Selection Rubric
```markdown
## Resource Selection Priority

1. **Contract Violations** (Critical)
   - Missing v2.0 requirements
   - Broken health checks
   - Failed lifecycle hooks
   - CLI non-functional

2. **Stability Issues** (High)
   - Crashes or hangs
   - Memory leaks
   - Connection failures
   - Data loss risks

3. **Integration Problems** (Medium-high)
   - Broken scenario dependencies
   - API incompatibilities
   - Configuration issues
   - Port conflicts

4. **Enhancement Opportunities** (Medium)
   - Performance improvements
   - New capabilities
   - Better error handling
   - Monitoring additions

5. **Polish** (Low)
   - Documentation updates
   - Code cleanup
   - Logging improvements
   - UI refinements
```

## Dynamic Selection Adjustment

### Learning from Outcomes
```python
def update_selection_weights(completed_item):
    """
    Adjust selection weights based on outcomes
    """
    if completed_item['result']['status'] == 'success':
        # Increase weight for similar items
        similar_type_boost = 1.1
        similar_target_boost = 1.05
    else:
        # Decrease weight temporarily
        similar_type_penalty = 0.9
        similar_target_penalty = 0.85
    
    # Apply adjustments to future selections
    update_scoring_weights(completed_item['type'], 
                          similar_type_boost)
    update_scoring_weights(completed_item['target'], 
                          similar_target_boost)
```

### Time-Based Adjustments
```python
def adjust_for_time_of_day():
    """
    Adjust selection based on time constraints
    """
    current_hour = datetime.now().hour
    
    if current_hour < 10:  # Morning - fresh mind
        return "complex"  # Tackle hard problems
    elif current_hour < 14:  # Midday - peak productivity
        return "valuable"  # Maximum value work
    elif current_hour < 17:  # Afternoon - waning focus
        return "quick"  # Quick wins
    else:  # Evening - wrap up
        return "cleanup"  # Documentation, small fixes
```

## Selection Monitoring

### Track Selection Effectiveness
```python
def monitor_selection_performance():
    """
    Monitor how well selection algorithm performs
    """
    metrics = {
        'success_rate': calculate_success_rate(),
        'value_delivered': sum_completed_value(),
        'average_time_accuracy': compare_estimated_vs_actual(),
        'collision_rate': count_worker_conflicts(),
        'blocker_resolution_time': average_blocker_wait()
    }
    
    # Adjust algorithm if metrics are poor
    if metrics['success_rate'] < 0.7:
        increase_success_probability_weight()
    if metrics['collision_rate'] > 0.1:
        increase_cooldown_periods()
    
    return metrics
```

## Selection Commands

### Manual Selection Tools
```bash
# Recommend top items with cooldown
recommend_items() {
    local k=${1:-5}  # Top K items
    local goals=${2:-"general improvement"}
    
    echo "Top $k recommendations for: $goals"
    
    # Score all available items
    for item in queue/pending/*.yaml; do
        if is_available $item; then
            score=$(calculate_score $item "$goals")
            echo "$score $item"
        fi
    done | sort -rn | head -n $k
}

# Record selection for tracking
record_selection() {
    local item=$1
    local agent=${2:-$USER}
    
    echo "{\"timestamp\": \"$(date -Iseconds)\", \
           \"agent\": \"$agent\", \
           \"selected\": \"$item\", \
           \"reason\": \"$3\"}" >> selections.jsonl
}
```

## Remember for Selection

**Value over volume** - Better to complete one high-value item than three low-value ones

**Avoid collisions** - Respect cooldowns and check other workers

**Learn from patterns** - Successful selections should influence future choices

**Balance the portfolio** - Mix quick wins with important long-term work

**Track effectiveness** - Monitor if selections achieve intended value

Good selection is half the battle. Choose work that advances the system most effectively while avoiding conflicts and maintaining steady progress.