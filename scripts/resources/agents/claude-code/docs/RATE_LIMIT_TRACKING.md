# Rate Limit Tracking System

## Overview

The Claude Code resource includes a sophisticated rate limit detection and usage tracking system to help prevent hitting API limits and provide visibility into usage patterns.

## Accuracy Considerations

### Known Limitations

1. **Estimated Limits**: The limits are **estimates** based on publicly available information about Claude Code's weekly hour allocations:
   - Pro: 40-80 hours/week of Sonnet 4
   - Max $100: 140-280 hours/week of Sonnet 4
   - Max $200: 240-480 hours/week of Sonnet 4
   
   These hour allocations have been converted to approximate request counts, but actual limits vary based on message length and complexity.

2. **Rolling Window**: The 5-hour limit uses a **rolling window**, not fixed reset times. We estimate the reset time based on the oldest request in the current 5-hour window, but this is an approximation.

3. **Conflicting Information**: Sources vary on whether the reset period is 5 hours or 8 hours. We default to 5 hours based on most recent documentation.

### Accuracy Improvements

To maintain accuracy, the system:

1. **Tracks Source Information**: Records where limits came from and when they were last verified
2. **Auto-Calibration**: Attempts to extract actual limit values from error messages
3. **User Confirmation**: Updates verification date when users manually set their tier
4. **Observed Values**: Stores observed rate limits separately from estimates
5. **Staleness Detection**: Warns when limits haven't been verified in >30 days

## Usage

### Check Current Usage
```bash
resource-claude-code usage
```

Shows:
- Current usage (hourly, 5-hour rolling, daily, weekly)
- Usage percentages
- Time until reset (estimated)
- Source information and accuracy warnings

### Set Subscription Tier
```bash
resource-claude-code set-tier pro    # Options: free, pro, max_100, max_200
```

This updates the limits and records that you've confirmed your tier.

### Reset Counters (Testing)
```bash
resource-claude-code reset-usage all  # Options: hourly, daily, weekly, all
```

## Implementation Details

### Files

- **Usage Tracking**: `~/.claude/usage_tracking.json`
- **Functions**: `lib/common.sh` (tracking functions)
- **Integration**: `lib/execute.sh` (request tracking, limit detection)
- **Display**: `cli.sh` (usage command)

### Data Structure

```json
{
  "hourly_requests": {"2024012514": 5, ...},
  "daily_requests": {"20240125": 45, ...},
  "weekly_requests": {"202404": 234, ...},
  "rate_limit_encounters": [...],
  "subscription_tier": "pro",
  "estimated_limits": {
    "pro": {"5_hour": 45, "daily": 216, "weekly": 1500}
  },
  "limits_source": {
    "description": "Estimates based on Claude Code weekly hour limits",
    "sources": [...],
    "last_verified": "2025-01-21",
    "accuracy_note": "These are ESTIMATES. Actual limits vary.",
    "rolling_window": "5-hour limit uses rolling window"
  }
}
```

### Key Functions

- `claude_code::detect_rate_limit()` - Pattern matching for rate limit errors
- `claude_code::track_request()` - Records each request
- `claude_code::get_usage()` - Retrieves current statistics
- `claude_code::check_usage_limits()` - Warns at 80% and 95% thresholds
- `claude_code::time_until_reset()` - Estimates reset times (approximate for 5-hour)
- `claude_code::update_observed_limit()` - Auto-calibration from observed limits

## Future Improvements

### TODO: LiteLLM Fallback
When rate limits are hit, automatically fallback to LiteLLM:
1. Check if LiteLLM resource is available
2. Switch execution backend to LiteLLM API
3. Convert Claude prompt format to LiteLLM format
4. Manage fallback state and recovery

See TODO comments in:
- `lib/common.sh:check_usage_limits()` lines 349-354, 361-362
- `lib/execute.sh:claude_code::run()` lines 79-83
- `lib/execute.sh` rate limit handler lines 185-189

### Potential Enhancements

1. **API Integration**: If Anthropic provides a usage API, integrate directly
2. **Machine Learning**: Learn actual limits from patterns over time
3. **Multi-User Support**: Track limits per API key/account
4. **Webhooks**: Send alerts when approaching limits
5. **Historical Analysis**: Predict usage patterns based on history

## References

- [Anthropic Help: Using Claude Code with Pro/Max](https://support.anthropic.com/en/articles/11145838)
- [TechCrunch: Anthropic unveils new rate limits (2025-07-28)](https://techcrunch.com/2025/07/28/anthropic-unveils-new-rate-limits-to-curb-claude-code-power-users/)
- [Claude Code Weekly Rate Limits Discussion](https://news.ycombinator.com/item?id=44713757)

Last Updated: 2025-08-21