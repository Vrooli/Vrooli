# Audio Tools CLI Documentation

## Installation

The Audio Tools CLI is installed automatically when you set up the scenario:
```bash
make setup
# OR
vrooli scenario setup audio-tools
```

The CLI will be available as `audio-tools` in your PATH.

## Configuration

### Initial Setup
```bash
audio-tools configure
```

This creates a configuration file at `~/.audio-tools/config.json` with default settings.

### Configuration Options
```json
{
  "api_url": "http://localhost:PORT",
  "default_format": "mp3",
  "default_quality": "high",
  "batch_size": 10,
  "output_directory": "./output",
  "verbose": false
}
```

### View Configuration
```bash
audio-tools configure --get api_url
audio-tools configure --list
```

### Update Configuration
```bash
audio-tools configure --set api_url http://localhost:8120
audio-tools configure --set default_format flac
audio-tools configure --set verbose true
```

## Global Options

All commands support these global options:
- `--help, -h`: Show help information
- `--version, -v`: Display version information
- `--verbose`: Enable verbose output
- `--quiet, -q`: Suppress non-error output
- `--config PATH`: Use alternative config file
- `--api-url URL`: Override API URL
- `--output DIR, -o DIR`: Specify output directory

## Commands

### Status Commands

#### Check Service Status
```bash
audio-tools status
```
Shows API health, version, and connection status.

#### List Processed Files
```bash
audio-tools list
audio-tools list --limit 10
audio-tools list --format json
```

### File Operations

#### Convert Format
```bash
# Basic conversion
audio-tools convert input.wav output.mp3

# With quality settings
audio-tools convert input.wav output.mp3 --quality high --bitrate 320k

# Batch conversion
audio-tools convert *.wav --format mp3 --output-dir ./converted

# Preserve original metadata
audio-tools convert input.flac output.mp3 --preserve-metadata
```

**Supported Formats:**
- MP3 (`.mp3`)
- WAV (`.wav`) 
- FLAC (`.flac`)
- AAC (`.aac`, `.m4a`)
- OGG Vorbis (`.ogg`)

#### Extract Metadata
```bash
# Display metadata
audio-tools metadata song.mp3

# Export to JSON
audio-tools metadata song.mp3 --json > metadata.json

# Batch extraction
audio-tools metadata *.mp3 --output metadata.csv
```

### Audio Editing

#### Trim Audio
```bash
# Trim from 10s to 30s
audio-tools trim input.mp3 output.mp3 --start 10 --end 30

# Trim first 30 seconds
audio-tools trim input.mp3 output.mp3 --duration 30

# Remove first 5 seconds
audio-tools trim input.mp3 output.mp3 --start 5
```

#### Adjust Volume
```bash
# Increase volume by 50%
audio-tools volume input.mp3 output.mp3 --level 1.5

# Decrease volume by 25%
audio-tools volume input.mp3 output.mp3 --level 0.75

# Normalize to target level
audio-tools normalize input.mp3 output.mp3 --target -16
```

#### Add Fade Effects
```bash
# Add 3-second fade in
audio-tools fade-in input.mp3 output.mp3 --duration 3

# Add 5-second fade out
audio-tools fade-out input.mp3 output.mp3 --duration 5

# Add both fades
audio-tools fade input.mp3 output.mp3 --in 3 --out 5
```

#### Merge Audio Files
```bash
# Concatenate files
audio-tools merge file1.mp3 file2.mp3 output.mp3

# Merge multiple files
audio-tools merge *.mp3 --output merged.mp3

# With crossfade
audio-tools merge file1.mp3 file2.mp3 --crossfade 2 --output merged.mp3
```

#### Split Audio
```bash
# Split at specific points (in seconds)
audio-tools split input.mp3 --points 30,60,90

# Split into equal chunks
audio-tools split input.mp3 --chunks 4

# Split by silence
audio-tools split input.mp3 --by-silence --threshold -40
```

### Audio Enhancement

#### Normalize Audio
```bash
# Basic normalization
audio-tools normalize input.mp3 output.mp3

# With specific target level
audio-tools normalize input.mp3 output.mp3 --level -16

# Prevent clipping
audio-tools normalize input.mp3 output.mp3 --no-clip
```

#### Enhance Quality
```bash
# Apply all enhancements
audio-tools enhance input.mp3 output.mp3

# Specific enhancements
audio-tools enhance input.mp3 output.mp3 --noise-reduction 0.8
audio-tools enhance input.mp3 output.mp3 --auto-level
audio-tools enhance input.mp3 output.mp3 --remove-clicks
```

#### Apply Noise Reduction
```bash
# Basic noise reduction
audio-tools denoise input.mp3 output.mp3

# With intensity setting (0.0-1.0)
audio-tools denoise input.mp3 output.mp3 --intensity 0.9
```

### Speed and Pitch

#### Change Speed
```bash
# Speed up by 1.5x
audio-tools speed input.mp3 output.mp3 --factor 1.5

# Slow down to 0.75x
audio-tools speed input.mp3 output.mp3 --factor 0.75

# Preserve pitch
audio-tools speed input.mp3 output.mp3 --factor 1.5 --preserve-pitch
```

#### Change Pitch
```bash
# Raise pitch by 2 semitones
audio-tools pitch input.mp3 output.mp3 --shift 2

# Lower pitch by 3 semitones
audio-tools pitch input.mp3 output.mp3 --shift -3

# Preserve tempo
audio-tools pitch input.mp3 output.mp3 --shift 2 --preserve-tempo
```

### Equalization

#### Apply EQ
```bash
# Apply preset EQ
audio-tools eq input.mp3 output.mp3 --preset bass-boost
audio-tools eq input.mp3 output.mp3 --preset vocal

# Custom EQ bands (frequency:gain)
audio-tools eq input.mp3 output.mp3 --bands 60:3,250:0,1000:-2,4000:1.5,16000:2
```

**Available Presets:**
- `bass-boost`: Enhance low frequencies
- `treble-boost`: Enhance high frequencies
- `vocal`: Optimize for voice
- `flat`: Reset to neutral
- `loudness`: V-shaped curve

### Voice Processing

#### Voice Activity Detection
```bash
# Detect speech segments
audio-tools vad recording.mp3

# With custom threshold
audio-tools vad recording.mp3 --threshold -35

# Export segments to JSON
audio-tools vad recording.mp3 --json > segments.json
```

#### Remove Silence
```bash
# Basic silence removal
audio-tools remove-silence input.mp3 output.mp3

# With custom threshold
audio-tools remove-silence input.mp3 output.mp3 --threshold -35

# Minimum silence duration
audio-tools remove-silence input.mp3 output.mp3 --min-duration 0.5
```

### Analysis

#### Analyze Audio
```bash
# Basic analysis
audio-tools analyze input.mp3

# Detailed analysis with spectrum
audio-tools analyze input.mp3 --detailed

# Export to JSON
audio-tools analyze input.mp3 --json > analysis.json

# Compare multiple files
audio-tools analyze file1.mp3 file2.mp3 --compare
```

### Batch Processing

#### Process Multiple Files
```bash
# Convert all WAV files to MP3
audio-tools batch convert *.wav --format mp3

# Apply same operation to multiple files
audio-tools batch normalize *.mp3 --level -16

# Process with custom pattern
audio-tools batch enhance "podcast*.mp3" --noise-reduction 0.8

# Parallel processing
audio-tools batch convert *.wav --format mp3 --parallel 4
```

#### Using Process Files
Create a process file `batch.txt`:
```
convert input1.wav output1.mp3 --quality high
trim input2.mp3 output2.mp3 --start 10 --end 60
enhance input3.mp3 output3.mp3 --noise-reduction 0.9
```

Run batch:
```bash
audio-tools batch --file batch.txt
audio-tools batch --file batch.txt --parallel 2
```

## Advanced Features

### Pipeline Processing
Chain multiple operations:
```bash
# Using pipe operator
audio-tools trim input.mp3 - --start 10 --end 60 | \
audio-tools normalize - - --level -16 | \
audio-tools fade - output.mp3 --in 2 --out 3

# Using chain command
audio-tools chain input.mp3 output.mp3 \
  --trim start=10,end=60 \
  --normalize level=-16 \
  --fade in=2,out=3
```

### Templates
Save and reuse processing templates:
```bash
# Save template
audio-tools template save podcast-process \
  --normalize level=-16 \
  --enhance noise-reduction=0.8 \
  --eq preset=vocal

# Use template
audio-tools template apply podcast-process input.mp3 output.mp3

# List templates
audio-tools template list

# Show template details
audio-tools template show podcast-process
```

### Watch Mode
Monitor directory for automatic processing:
```bash
# Watch for new files
audio-tools watch ./uploads --format mp3 --output ./processed

# With specific pattern
audio-tools watch ./recordings --pattern "*.wav" \
  --normalize --output ./normalized
```

## Configuration Examples

### Podcast Processing Setup
```bash
audio-tools configure --set default_quality high
audio-tools configure --set default_format mp3
audio-tools template save podcast \
  --normalize level=-16 \
  --enhance noise-reduction=0.8,auto-level=true \
  --eq preset=vocal
```

### Music Mastering Setup
```bash
audio-tools template save master \
  --normalize level=-14,prevent-clipping=true \
  --eq bands=60:1,250:0,1000:0,4000:1,16000:1 \
  --enhance auto-level=true
```

## Error Handling

### Common Issues

#### API Connection Failed
```bash
# Check status
audio-tools status

# Update API URL
audio-tools configure --set api_url http://localhost:8120
```

#### Unsupported Format
```bash
# Check supported formats
audio-tools formats

# Force format detection
audio-tools convert input.audio output.mp3 --force
```

#### File Not Found
```bash
# Use absolute path
audio-tools convert /full/path/to/file.wav output.mp3

# Check file exists
audio-tools check input.mp3
```

## Output Formats

### JSON Output
```bash
audio-tools metadata input.mp3 --json
audio-tools analyze input.mp3 --json
audio-tools vad input.mp3 --json
```

### CSV Output
```bash
audio-tools batch analyze *.mp3 --csv > analysis.csv
audio-tools metadata *.mp3 --csv > metadata.csv
```

### Verbose Output
```bash
audio-tools convert input.wav output.mp3 --verbose
```

## Performance Tips

1. **Batch Processing**: Use batch commands for multiple files
2. **Parallel Execution**: Use `--parallel N` for concurrent processing
3. **Output Directory**: Specify output directory to avoid overwriting
4. **Templates**: Create templates for repeated workflows
5. **Pipeline**: Chain operations to avoid intermediate files
6. **Format Selection**: Use lossless formats for intermediate processing

## Integration with Other Tools

### With file-tools
```bash
file-tools find . -name "*.wav" | \
  xargs -I {} audio-tools convert {} {}.mp3
```

### With data-tools
```bash
audio-tools analyze *.mp3 --json | \
  data-tools aggregate --by duration
```

### In Scripts
```bash
#!/bin/bash
for file in *.wav; do
  audio-tools convert "$file" "${file%.wav}.mp3" \
    --quality high \
    --preserve-metadata
done
```

## Shortcuts and Aliases

Add to your shell configuration:
```bash
# ~/.bashrc or ~/.zshrc
alias at='audio-tools'
alias atc='audio-tools convert'
alias atn='audio-tools normalize'
alias ate='audio-tools enhance'
alias atv='audio-tools vad'

# Common workflows
alias podcast-process='audio-tools template apply podcast'
alias music-master='audio-tools template apply master'
```

## Environment Variables

```bash
# Override default settings
export AUDIO_TOOLS_API_URL="http://localhost:8120"
export AUDIO_TOOLS_OUTPUT_DIR="./processed"
export AUDIO_TOOLS_DEFAULT_FORMAT="flac"
export AUDIO_TOOLS_VERBOSE="true"
```

## Exit Codes

- `0`: Success
- `1`: General error
- `2`: Invalid arguments
- `3`: File not found
- `4`: API connection error
- `5`: Unsupported format
- `6`: Processing failed
- `127`: Command not found

## Getting Help

```bash
# General help
audio-tools --help

# Command-specific help
audio-tools convert --help
audio-tools enhance --help

# Show examples
audio-tools examples

# Show version
audio-tools --version
```