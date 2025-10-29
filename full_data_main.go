package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "strings"
    "time"
)

type SymbolicConfig struct {
    Prefix            string  `json:"prefix"`
    KG                string  `json:"KG"`
    RulesFile         string  `json:"rules_file"`
    RdfFile           string  `json:"rdf_file"`
    ConstraintsFolder string  `json:"constraints_folder"`
    PCAThreshold      float64 `json:"pca_threshold"`
    SkipValidation    bool    `json:"skip_validation,omitempty"`
}

// DataFrame representation
type DataFrame struct {
    Columns []string               `json:"columns"`
    Data    []map[string]interface{} `json:"data"`
    Shape   []int                  `json:"shape"`
    Dtypes  map[string]string      `json:"dtypes"`
}

// Triple representation
type Triple struct {
    Subject    string `json:"subject"`
    Predicate  string `json:"predicate"`
    Object     string `json:"object"`
    ObjectType string `json:"object_type,omitempty"`
}

// Graph representation
type GraphData struct {
    Triples      []Triple          `json:"triples"`
    TotalTriples int               `json:"total_triples"`
    Namespaces   map[string]string `json:"namespaces"`
    LimitedTo    *int              `json:"limited_to"`
}

// SPARQL Query info
type QueryInfo struct {
    Query         string  `json:"query"`
    ExecutionTime float64 `json:"execution_time"`
    ResultCount   int     `json:"result_count"`
    Timestamp     string  `json:"timestamp"`
}

// Complete result structure
type FullDataResult struct {
    Success         bool      `json:"success"`
    ExecutionTime   float64   `json:"execution_time"`
    Timestamp       string    `json:"timestamp"`

    // Predictions as DataFrame
    PredictionsDataframe DataFrame `json:"predictions_dataframe"`

    // New triples only
    NewTriples []Triple `json:"new_triples"`

    // Graph data
    Graphs struct {
        Initial GraphData `json:"initial"`
        Enriched GraphData `json:"enriched"`
        Statistics struct {
            InitialTriples   int `json:"initial_triples"`
            EnrichedTriples  int `json:"enriched_triples"`
            PredictionsAdded int `json:"predictions_added"`
        } `json:"statistics"`
    } `json:"graphs"`

    // Queries
    SPARQLQueries []QueryInfo `json:"sparql_queries"`

    // Summary
    Summary struct {
        TotalPredictions     int  `json:"total_predictions"`
        QueriesExecuted      int  `json:"queries_executed"`
        ProcessingSuccessful bool `json:"processing_successful"`
    } `json:"summary"`

    Error string `json:"error,omitempty"`
}

func callFullDataWrapper(config SymbolicConfig) (*FullDataResult, error) {
    configJSON, err := json.Marshal(config)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal config: %w", err)
    }

    cmd := exec.Command("python3", "full_data_wrapper.py")
    cmd.Stdin = bytes.NewReader(configJSON)

    var out bytes.Buffer
    cmd.Stdout = &out

    // Capture stderr for debugging
    var stderr bytes.Buffer
    cmd.Stderr = &stderr

    fmt.Printf("Processing %s with full data capture...\n", config.KG)
    err = cmd.Run()

    // If there was an error, show stderr
    if err != nil {
        fmt.Printf("Error executing Python: %v\n", err)
        if stderr.Len() > 0 {
            fmt.Printf("Python stderr:\n%s\n", stderr.String())
        }
    }

    var result FullDataResult
    if err := json.Unmarshal(out.Bytes(), &result); err != nil {
        return nil, fmt.Errorf("failed to parse output: %w\nRaw output: %s", err, out.String())
    }

    if err != nil && !result.Success {
        return &result, fmt.Errorf("processing failed: %s", result.Error)
    }

    return &result, nil
}

func analyzeResults(result *FullDataResult) {
    fmt.Println("\n📊 FULL DATA ANALYSIS")
    fmt.Println(strings.Repeat("=", 50))

    // DataFrame analysis
    fmt.Printf("\n📋 Predictions DataFrame:\n")
    if len(result.PredictionsDataframe.Shape) >= 2 {
        fmt.Printf("   Shape: %dx%d\n", result.PredictionsDataframe.Shape[0], result.PredictionsDataframe.Shape[1])
    }
    fmt.Printf("   Columns: %v\n", result.PredictionsDataframe.Columns)

    // Show sample predictions
    if len(result.NewTriples) > 0 {
        fmt.Printf("\n🔮 Sample Predictions (first 5):\n")
        count := 5
        if len(result.NewTriples) < 5 {
            count = len(result.NewTriples)
        }
        for i := 0; i < count; i++ {
            triple := result.NewTriples[i]
            fmt.Printf("   %s -[%s]-> %s\n", triple.Subject, triple.Predicate, triple.Object)
        }
        if len(result.NewTriples) > 5 {
            fmt.Printf("   ... and %d more predictions\n", len(result.NewTriples)-5)
        }
    } else {
        fmt.Printf("\n⚠️  No predictions generated\n")
    }

    // Graph statistics
    fmt.Printf("\n📈 Graph Statistics:\n")
    fmt.Printf("   Initial triples: %d\n", result.Graphs.Statistics.InitialTriples)
    fmt.Printf("   Enriched triples: %d\n", result.Graphs.Statistics.EnrichedTriples)
    fmt.Printf("   New predictions: %d\n", result.Graphs.Statistics.PredictionsAdded)

    // Query analysis
    fmt.Printf("\n🔍 SPARQL Queries:\n")
    fmt.Printf("   Total executed: %d\n", len(result.SPARQLQueries))

    totalResults := 0
    for _, query := range result.SPARQLQueries {
        totalResults += query.ResultCount
    }
    fmt.Printf("   Total results: %d\n", totalResults)

    // Show a sample query if available
    if len(result.SPARQLQueries) > 0 {
        fmt.Printf("\n   Sample query:\n")
        query := result.SPARQLQueries[0]
        lines := strings.Split(query.Query, "\n")
        for _, line := range lines {
            if len(strings.TrimSpace(line)) > 0 {
                fmt.Printf("     %s\n", line)
            }
        }
        fmt.Printf("   Results: %d, Time: %.3fs\n", query.ResultCount, query.ExecutionTime)
    }

    // Performance
    fmt.Printf("\n⏱️  Performance:\n")
    fmt.Printf("   Execution time: %.2f seconds\n", result.ExecutionTime)
}

func saveResultsToFile(result *FullDataResult, filename string) error {
    data, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(filename, data, 0644)
}

func processMultipleKGs(configs []SymbolicConfig) {
    var allPredictions []Triple
    totalQueries := 0
    successCount := 0

    fmt.Printf("\n🔄 Processing %d Knowledge Graphs\n", len(configs))
    fmt.Println(strings.Repeat("-", 50))

    for i, config := range configs {
        fmt.Printf("\n[%d/%d] Processing %s...\n", i+1, len(configs), config.KG)

        result, err := callFullDataWrapper(config)
        if err != nil {
            fmt.Printf("❌ Error: %v\n", err)
            continue
        }

        if result.Success {
            successCount++
            allPredictions = append(allPredictions, result.NewTriples...)
            totalQueries += len(result.SPARQLQueries)

            // Save individual results
            filename := fmt.Sprintf("results_%s_%s.json", config.KG, time.Now().Format("20060102_150405"))
            if err := saveResultsToFile(result, filename); err == nil {
                fmt.Printf("✅ Saved results to %s\n", filename)
            }

            fmt.Printf("   Generated %d predictions\n", len(result.NewTriples))
        }
    }

    fmt.Printf("\n📊 Aggregate Results:\n")
    fmt.Println(strings.Repeat("-", 50))
    fmt.Printf("   Successful KGs: %d/%d\n", successCount, len(configs))
    fmt.Printf("   Total predictions: %d\n", len(allPredictions))
    fmt.Printf("   Total queries: %d\n", totalQueries)
}

func main() {
    fmt.Println("🚀 Full Data Symbolic Predictions Processor")
    fmt.Println(strings.Repeat("=", 50))

    // Check if wrapper exists
    if _, err := os.Stat("full_data_wrapper.py"); os.IsNotExist(err) {
        fmt.Println("❌ Error: full_data_wrapper.py not found")
        fmt.Println("Please create the wrapper file first")
        return
    }

    config := SymbolicConfig{
        Prefix:            "http://FrenchRoyalty.org/", // Note the typo to match RDF
        KG:                "FrenchRoyalty",
        RulesFile:         "french_royalty.csv",
        RdfFile:           "french_royalty.nt",
        ConstraintsFolder: "FrenchRoyalty",
        PCAThreshold:      0.7,
        SkipValidation:    true,
    }

    result, err := callFullDataWrapper(config)
    if err != nil {
        fmt.Printf("❌ Error: %v\n", err)
        return
    }

    if result.Success {
        analyzeResults(result)

        // Save complete results
        filename := fmt.Sprintf("full_results_%s.json", time.Now().Format("20060102_150405"))
        if err := saveResultsToFile(result, filename); err == nil {
            fmt.Printf("\n💾 Complete results saved to: %s\n", filename)
        }
    } else {
        fmt.Printf("❌ Processing failed: %s\n", result.Error)
    }
}