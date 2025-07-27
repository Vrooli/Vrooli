# Test Outputs Directory

This directory contains files generated during example execution and testing. It's organized to keep test outputs separate from clean example code.

## 📁 Directory Structure

- **screenshots/** - Generated screenshots from core automation examples
- **logs/** - Log files and execution traces from AI examples  
- **temp/** - Temporary files created during example execution

## 🔄 Usage

Examples automatically create and save outputs here:
- Screenshot examples save images to `screenshots/`
- AI examples may log interactions to `logs/`
- Temporary files during execution go to `temp/`

## 🧹 Cleanup

This directory is designed to be safely cleared:
```bash
# Clear all test outputs
rm -rf screenshots/* logs/* temp/*

# Or clear specific types
rm -f screenshots/*.png screenshots/*.jpg
rm -f logs/*.log
rm -f temp/*
```

## 📝 Git Behavior

All files in this directory are ignored by git (see `.gitignore`) except:
- This README.md
- The .gitignore file itself

This keeps the repository clean while allowing examples to generate outputs locally.