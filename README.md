# DynamicPipeline-KGC

A concurrent system for generating symbolic predictions from RDF knowledge graphs using rule-based reasoning, with Go orchestration for parallel processing.

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

### 1. Install requirements
pip install -r requirements.txt

### 2. Run the main program
go run full_data_main.go



