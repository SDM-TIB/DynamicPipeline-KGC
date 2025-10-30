# DynamicPipeline-KGC

A concurrent system for generating symbolic predictions from RDF knowledge graphs using rule-based reasoning, with Go orchestration for parallel processing.

## 📋 Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Directory Structure](#directory-structure)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Data Formats](#data-formats)
- [Troubleshooting](#troubleshooting)
- [Examples](#examples)

## 📂 Project Structure
```
symbolic-predictions-project/
├── full_data_main.go        # Main Go program (START HERE)
├── full_data_wrapper.py     # Python wrapper for data processing
├── Symbolic_predictions.py  # Core prediction engine
├── KG/                      # Knowledge graph data
├── Rules/                   # Rule files with confidence scores
└── logs/                    # Execution logs
```

## Execution Steps

### 1. Navigate to project directory
cd /path/to/symbolic-predictions-project

### 2. Activate Python environment
source symbolic_env/bin/activate

### 3. Run the main program
go run full_data_main.go


